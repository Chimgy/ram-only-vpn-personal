package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"n-api/authCrypto"
	"n-api/peerpool"
	"n-api/wg"
)

// check and load in crucial config for connections
type Config struct {
	APIKey    string
	DuckToken string
	Domain    string
	StaticIP  string
	Port      string
}

// global var to clean up code
var cfg Config

// need to parse /etc/n-api/config.env
func loadEnvFile(filename string) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return // file doesnt exist
	}
	for _, line := range strings.Split(string(content), "\n") {
		pair := strings.SplitN(line, "=", 2)
		if len(pair) == 2 {
			os.Setenv(strings.TrimSpace(pair[0]), strings.TrimSpace(pair[1]))
		}
	}
}

func loadConfig() {
	loadEnvFile("/etc/n-api/config.env")
	cfg = Config{
		APIKey:    os.Getenv("NODE_API_KEY"),
		DuckToken: os.Getenv("DUCKDNS_TOKEN"),
		Domain:    os.Getenv("DUCKDNS_DOMAIN"),
		StaticIP:  os.Getenv("STATIC_IP"),
		Port:      os.Getenv("API_PORT"),
	}
	if cfg.Domain == "" && cfg.StaticIP == "" {
		log.Println("WARNING: Neither DuckDNS domain nor STATIC_IP set — clients won't receive a valid endpoint")
	}
	if cfg.APIKey == "" {
		log.Fatal("NODE_API_KEY not set")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
}

var pool *peerpool.Pool

// Request/response types

type addPeerRequest struct {
	PublicKey string `json:"public_key"`
}

type addPeerResponse struct {
	TunnelIP       string `json:"tunnel_ip"`
	ServerPubkey   string `json:"server_pubkey"`
	ServerEndpoint string `json:"server_endpoint"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func getPublicIP() string {
	client := &http.Client{Timeout: 5 * time.Second}
	// try 5 times because pi can be slow to boot
	for i := 0; i < 5; i++ {
		// Golang is so cool (it will find the ca-certs for us)
		resp, err := client.Get("https://ifconfig.me/ip")
		if err != nil {
			log.Printf("Attemp %d: network not ready, retrying...", i+1)
			time.Sleep(2 * time.Second)
			continue
		}
		ip, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err == nil {
			return strings.TrimSpace(string(ip))
		}
	}
	return "unavailable"
}

// DUCKDNS required for dynamic ips otherwise just set ur baseURL to the static one in vpn-client/api-go
func updateDuckDNS() {
	// Create a client because if http.Get() gets stuck it will wait forever apparently
	var client = &http.Client{
		Timeout: time.Second * 10,
	}

	if cfg.DuckToken == "" || cfg.Domain == "" {
		log.Println("DuckDns env vars not set, skipping update...")
		return
	}

	url := fmt.Sprintf("https://www.duckdns.org/update?domains=%s&token=%s", cfg.Domain, cfg.DuckToken)
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("DuckDNS update failed: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Println("DuckDNS updated successfully")
}

// Duckdns needs to be updated every now and again if you have a dynamic ip
func startDNSHeartbeat(interval time.Duration) {
	go func() {
		// initially update immediately on start up
		updateDuckDNS()

		for range time.Tick(interval) {
			updateDuckDNS()
		}
	}()
	log.Printf("DNS Heartbeat started: interval=%v", interval)
}

// Helpers

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

// POST /peer
// Body: { "public_key": "...", "user_id": "..." }
// Returns: { "tunnel_ip": "10.8.0.x", "server_pubkey": "...", "server_endpoint": "x.x.x.x:51820" }
func handleAddPeer(w http.ResponseWriter, r *http.Request, publicIP string) {
	var req addPeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	req.PublicKey = strings.TrimSpace(req.PublicKey)
	if req.PublicKey == "" {
		writeError(w, http.StatusBadRequest, "public_key required")
		return
	}

	// Assign tunnel IP (idempotent same pubkey gets same IP)
	tunnelIP, err := pool.Assign(req.PublicKey)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	// Add peer to live WireGuard interface
	if err := wg.AddPeer(req.PublicKey, tunnelIP.String()); err != nil {
		// Roll back pool assignment so IP isn't leaked
		pool.Release(req.PublicKey)
		log.Printf("wg AddPeer failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to configure WireGuard peer")
		return
	}

	serverPubkey, err := wg.ServerPubkey()
	if err != nil {
		log.Printf("WARNING: could not read server pubkey: %v", err)
		serverPubkey = "unavailable"
	}

	log.Printf("Peer added: user=%s pubkey=%s tunnel=%s", req.PublicKey[:8]+"...", tunnelIP)

	var endpoint string
	switch {
	case cfg.Domain != "":
		endpoint = fmt.Sprintf("%s.duckdns.org:51820", cfg.Domain)
	case cfg.StaticIP != "":
		endpoint = fmt.Sprintf("%s:51820", cfg.StaticIP)
	default:
		endpoint = "UNCONFIGURED:51820"
	}

	writeJSON(w, http.StatusOK, addPeerResponse{
		TunnelIP:       tunnelIP.String(),
		ServerPubkey:   serverPubkey,
		ServerEndpoint: endpoint,
	})
}

// While the web application is used for parsing webkeys will need this to read what
// browser sends
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

// Encrypts the api-key sent on intial connection, the node + client will use pre-shared keys
// and decrypt internally to verify. Sniffers will only see random numbers on initial connection (both sides)
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {

	// grab key baked into node for comparison
	cryptoKey := authCrypto.DeriveKey(cfg.APIKey)

	// Send only 404 header (rather than using the go net/http error for extra stealth)

	return func(w http.ResponseWriter, r *http.Request) {
		// DECRYPT SENDER KEY & VERIFY
		// read it:
		encryptedBody, err := io.ReadAll(r.Body)
		// Any errors or no key silent drop

		// using hijacker kills tcp quicker then port scrapers can fingerprint (usually)
		if err != nil || len(encryptedBody) == 0 {
			if hj, ok := w.(http.Hijacker); ok {
				conn, _, _ := hj.Hijack()
				conn.Close() // Kill the TCP connection immediately
				return
			}
			// if hijack fails
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// verify it:
		plaintext, err := authCrypto.Decrypt(encryptedBody, cryptoKey)
		if err != nil {
			if hj, ok := w.(http.Hijacker); ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
				return
			}
			// silent drop (wrong key or bad data)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// put plaintext back into the requestbody (now decrypted) for next handler to use (addPeer etc)
		r.Body = io.NopCloser(bytes.NewBuffer(plaintext))

		next(w, r)
	}
}

// DELETE /peer
// Body: { "public_key": "..." }
func handleRemovePeer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PublicKey string `json:"public_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := wg.RemovePeer(req.PublicKey); err != nil {
		log.Printf("wg RemovePeer failed: %v", err)
	}

	pool.Release(req.PublicKey)
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// ------------------------- DEBUG ENDPOINTS ---------------------------
// (leave commented if not using because they arent wrapped in auth
// // GET /peers debug endpoint, shows active peers + pool status
// // Now will also show wg handshkae timestamps
// func handleListPeers(w http.ResponseWriter, r *http.Request) {
// 	peers := pool.List()

// 	statuses, _ := wg.ShowDump()
// 	hsMap := make(map[string]time.Time)
// 	for _, s := range statuses {
// 		hsMap[s.PublicKey] = s.LastHandshake
// 	}

// 	type enrichedPeer struct {
// 		PublicKey     string `json:"public_key"`
// 		TunnelIP      string `json:"tunnel_ip"`
// 		LastHandshake string `json:"last_handshake"`
// 	}

// 	enriched := make([]enrichedPeer, 0, len(peers))
// 	for _, p := range peers {
// 		hs := "never"
// 		if t, ok := hsMap[p.PublicKey]; ok && !t.IsZero() {
// 			hs = t.UTC().Format(time.RFC3339)
// 		}
// 		enriched = append(enriched, enrichedPeer{
// 			PublicKey:     p.PublicKey,
// 			TunnelIP:      p.TunnelIP.String(),
// 			LastHandshake: hs,
// 		})
// 	}

// 	writeJSON(w, http.StatusOK, map[string]any{
// 		"active":    enriched,
// 		"available": pool.Available(),
// 	})
// }

// // GET /health
// func handleHealth(w http.ResponseWriter, r *http.Request) {
// 	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
// }

// startReaper polls WireGuard handshake times and reaps silent peers
func startReaper(ttl, interval time.Duration) {
	go func() {
		for range time.Tick(interval) {
			statuses, err := wg.ShowDump()
			if err != nil {
				log.Printf("reaper: wg dump failed: %v", err)
				continue
			}

			now := time.Now()
			for _, s := range statuses {
				dead := s.LastHandshake.IsZero() || now.Sub(s.LastHandshake) > ttl
				if !dead {
					continue
				}

				log.Printf("reaper: reaping %s (last handshake: %v)", s.PublicKey[:8]+"...", s.LastHandshake)

				if err := wg.RemovePeer(s.PublicKey); err != nil {
					log.Printf("reaper: remove failed: %v", err)
				}
				pool.Release(s.PublicKey)
			}
		}
	}()
	log.Printf("reaper started:	ttl=%v poll=%v", ttl, interval)
}

func main() {
	loadConfig()

	publicIP := getPublicIP()
	log.Printf("Public IP: %s", publicIP)

	var err error
	// Pool: 10.8.0.2 — 10.8.0.50 (48 concurrent peers, expand as needed)
	pool, err = peerpool.New(2, 50)
	if err != nil {
		log.Fatalf("Failed to init peer pool: %v", err)
	}

	// reap peers silent for 3 minutes, check every 30 seconds
	startReaper(3*time.Minute, 30*time.Second)

	// Refresh DNS every (30) minutes
	if cfg.DuckToken != "" && cfg.Domain != "" {
		startDNSHeartbeat(15 * time.Minute)
	}

	http.HandleFunc("/peer", authMiddleware(corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleAddPeer(w, r, publicIP)
		case http.MethodDelete:
			handleRemovePeer(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "POST or DELETE only")
		}
	})))

	// Debug Endpoints:
	// http.HandleFunc("/peers", corsMiddleware(handleListPeers))
	// http.HandleFunc("/health", corsMiddleware(handleHealth))

	log.Printf("vpnode-api listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}

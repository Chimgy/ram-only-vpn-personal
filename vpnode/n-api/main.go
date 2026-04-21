package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"n-api/peerpool"
	"n-api/wg"
)

var pool *peerpool.Pool

// Request/response types

type addPeerRequest struct {
	PublicKey string `json:"public_key"`
	UserID    string `json:"user_id"`
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
	resp, err := http.Get("https://ifconfig.me")
	if err != nil {
		log.Printf("WARNING: could not fetch public IP: %v", err)
		return "unavailable"
	}
	defer resp.Body.Close()
	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "unavailable"
	}
	return strings.TrimSpace(string(ip))
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

	// Assign tunnel IP (idempotent — same pubkey gets same IP)
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

	log.Printf("Peer added: user=%s pubkey=%s tunnel=%s", req.UserID, req.PublicKey[:8]+"...", tunnelIP)

	writeJSON(w, http.StatusOK, addPeerResponse{
		TunnelIP:       tunnelIP.String(),
		ServerPubkey:   serverPubkey,
		ServerEndpoint: fmt.Sprintf("%s:51820", publicIP),
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

func apiKeyMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := os.Getenv("NODE_API_KEY")
		if apiKey != "" && r.Header.Get("X-API-Key") != apiKey {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
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

// GET /peers — debug endpoint, shows active peers + pool status
// Now will also show wg handshkae timestamps
func handleListPeers(w http.ResponseWriter, r *http.Request) {
	peers := pool.List()

	statuses, _ := wg.ShowDump()
	hsMap := make(map[string]time.Time)
	for _, s := range statuses {
		hsMap[s.PublicKey] = s.LastHandshake
	}

	type enrichedPeer struct {
		PublicKey     string `json:"public_key"`
		TunnelIP      string `json:"tunnel_ip"`
		LastHandshake string `json:"last_handshake"`
	}

	enriched := make([]enrichedPeer, 0, len(peers))
	for _, p := range peers {
		hs := "never"
		if t, ok := hsMap[p.PublicKey]; ok && !t.IsZero() {
			hs = t.UTC().Format(time.RFC3339)
		}
		enriched = append(enriched, enrichedPeer{
			PublicKey:     p.PublicKey,
			TunnelIP:      p.TunnelIP.String(),
			LastHandshake: hs,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"active":    enriched,
		"available": pool.Available(),
	})
}

// GET /health
func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

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

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/peer", corsMiddleware(apiKeyMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleAddPeer(w, r, publicIP)
		case http.MethodDelete:
			handleRemovePeer(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "POST or DELETE only")
		}
	})))

	http.HandleFunc("/peers", corsMiddleware(apiKeyMiddleware(handleListPeers)))
	http.HandleFunc("/health", corsMiddleware(apiKeyMiddleware(handleHealth)))

	log.Printf("vpnode-api listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

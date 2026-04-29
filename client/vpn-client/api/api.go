package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"vpn-client/authCrypto"
)

// // LAN OPTION:
// const baseURL = "http://<IP.OF.YOUR.PI>:8080"
// DUCKDNS Option (Default) for dynamic IP's
const baseURL = "http://ram-only-vpn.duckdns.org:8080"

// Static IP option / or if you have dynamic just change everytime
// const baseURL = "http://<static.or.dynamic.ip>:8080"

// IP option

// This is set within vpn-boot.sh and here. This is just a personal version so I honestly don't see the need to make it more l33t
const apiKey = "test123"

type PeerResponse struct {
	TunnelIP       string `json:"tunnel_ip"`
	ServerPubkey   string `json:"server_pubkey"`
	ServerEndpoint string `json:"server_endpoint"`
}

func Connect(publicKey, userID string) (PeerResponse, error) {
	data, _ := json.Marshal(map[string]string{
		"public_key": publicKey,
		"user_id":    userID,
	})

	// Encrypt the JSON (its being sent over http)
	key := authCrypto.DeriveKey(apiKey)
	encryptedData, _ := authCrypto.Encrypt(data, key)

	// Post the encrypted bytes
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/peer", bytes.NewReader(encryptedData))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)

	// handle response ()
	if err != nil {
		return PeerResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return PeerResponse{}, fmt.Errorf("authentication failed or node not found (404)")
	}

	if resp.StatusCode != http.StatusOK {
		return PeerResponse{}, fmt.Errorf("server error: %d", resp.StatusCode)
	}

	var pr PeerResponse
	err = json.NewDecoder(resp.Body).Decode(&pr)
	return pr, err
}

func Disconnect(publicKey string) error {
	// same method of encryption to disconnect
	data, _ := json.Marshal(map[string]string{"public_key": publicKey})
	key := authCrypto.DeriveKey(apiKey)
	encryptedData, _ := authCrypto.Encrypt(data, key)

	req, _ := http.NewRequest(http.MethodDelete, baseURL+"/peer", bytes.NewReader(encryptedData))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err // due to the 8080 connection silent dropping 404 will always be the result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to disconnect: %d", resp.StatusCode)
	}
	return nil
}

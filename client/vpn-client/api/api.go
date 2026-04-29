package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// THIS IS NOT DYNAMIC RN HAVE TO CHANGE BEFORE EVERY SINGLE BUILD
const baseURL = "http://192.168.1.107:8080"

// This is set within vpn-boot.sh and here for now before i figure out a more secure method for holding this
const apiKey = "test123"

type PeerResponse struct {
	TunnelIP       string `json:"tunnel_ip"`
	ServerPubkey   string `json:"server_pubkey"`
	ServerEndpoint string `json:"server_endpoint"`
}

func Connect(publicKey, userID string) (PeerResponse, error) {
	body, _ := json.Marshal(map[string]string{
		"public_key": publicKey,
		"user_id":    userID,
	})
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/peer", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return PeerResponse{}, fmt.Errorf("POST /peer: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return PeerResponse{}, fmt.Errorf("server returned %d", resp.StatusCode)
	}
	var pr PeerResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return PeerResponse{}, fmt.Errorf("decoding response: %w", err)
	}
	return pr, nil
}

func Disconnect(publicKey string) error {
	body, _ := json.Marshal(map[string]string{"public_key": publicKey})
	req, _ := http.NewRequest(http.MethodDelete, baseURL+"/peer", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE /peer: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}
	return nil
}

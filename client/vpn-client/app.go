package main

import (
	"context"
	"fmt"
	"sync"

	"vpn-client/api"
	"vpn-client/keypair"
	"vpn-client/tunnel"
)

type App struct {
	ctx       context.Context
	mu        sync.Mutex
	connected bool
	tunnelIP  string
	privKey   string
	pubKey    string
	apiKey    string
	baseURL   string
}

type StatusResult struct {
	Connected bool   `json:"connected"`
	TunnelIP  string `json:"tunnelIP"`
}

type ConnectResult struct {
	OK       bool   `json:"ok"`
	TunnelIP string `json:"tunnelIP"`
	Error    string `json:"error"`
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

func (a *App) Connect(apiKey, baseURL string) ConnectResult {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.connected {
		return ConnectResult{OK: false, Error: "already connected"}
	}

	kp, err := keypair.Generate()
	if err != nil {
		return ConnectResult{OK: false, Error: fmt.Sprintf("keypair: %v", err)}
	}

	peer, err := api.Connect(kp.PublicKey, apiKey, baseURL)
	if err != nil {
		return ConnectResult{OK: false, Error: fmt.Sprintf("api: %v", err)}
	}

	if err := tunnel.Up(kp.PrivateKey, peer.TunnelIP, peer.ServerPubkey, peer.ServerEndpoint); err != nil {
		_ = api.Disconnect(kp.PublicKey, apiKey, baseURL)
		return ConnectResult{OK: false, Error: fmt.Sprintf("tunnel: %v", err)}
	}

	a.connected = true
	a.tunnelIP = peer.TunnelIP
	a.privKey = kp.PrivateKey
	a.pubKey = kp.PublicKey
	a.baseURL = baseURL
	a.apiKey = apiKey

	return ConnectResult{OK: true, TunnelIP: peer.TunnelIP}
}

func (a *App) Disconnect() ConnectResult {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.connected {
		return ConnectResult{OK: false, Error: "not connected"}
	}

	if err := tunnel.Down(); err != nil {
		return ConnectResult{OK: false, Error: fmt.Sprintf("tunnel down: %v", err)}
	}

	_ = api.Disconnect(a.pubKey, a.apiKey, a.baseURL)

	a.connected = false
	a.tunnelIP = ""
	a.privKey = ""
	a.pubKey = ""
	a.baseURL = ""
	a.apiKey = ""

	return ConnectResult{OK: true}
}

func (a *App) GetStatus() StatusResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	return StatusResult{Connected: a.connected, TunnelIP: a.tunnelIP}
}

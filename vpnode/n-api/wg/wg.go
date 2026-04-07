package wg

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)


// PeerStatus holds live WireGuard state for a peer
type PeerStatus struct {
	PublicKey     string
	LastHandshake time.Time // zero if never
}

const iface = "wg0"

// AddPeer runs: wg set wg0 peer <pubkey> allowed-ips <tunnelIP>/32
// This adds the peer live without touching the config file
func AddPeer(pubkey, tunnelIP string) error {
	if err := validatePubkey(pubkey); err != nil {
		return err
	}

	cmd := exec.Command("wg", "set", iface,
		"peer", pubkey,
		"allowed-ips", tunnelIP+"/32",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wg set failed: %w — output: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

// RemovePeer runs: wg set wg0 peer <pubkey> remove
func RemovePeer(pubkey string) error {
	if err := validatePubkey(pubkey); err != nil {
		return err
	}

	cmd := exec.Command("wg", "set", iface,
		"peer", pubkey,
		"remove",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wg remove failed: %w — output: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

// ServerPubkey reads the current server public key from /run/wg/server.pub
// Which was written by vpn-boot.sh on every boot
func ServerPubkey() (string, error) {
	out, err := exec.Command("cat", "/run/wg/server.pub").Output()
	if err != nil {
		return "", fmt.Errorf("failed to read server pubkey: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// validatePubkey checks if pub key is valid, WireGuard pubkeys are 44-char base64
func validatePubkey(pubkey string) error {
	pubkey = strings.TrimSpace(pubkey)
	if len(pubkey) != 44 {
		return fmt.Errorf("invalid pubkey length %d (expected 44)", len(pubkey))
	}
	return nil
}

// ShowDump parses `wg show wg0 dump` and returns live peer statuses
// will be outputted as follows:
// interface line: <pubkey> <privkey> <listen-port> <fwmark>
// peer lines:     <pubkey> <preshared> <endpoint> <allowed-ips> <last-handshake-unix> <rx> <tx> <keepalive>

// will need to think about post dev set up for this:

func ShowDump() ([]PeerStatus, error) {
	out, err := exec.Command("wg", "show", iface, "dump").Output()
	if err != nil {
		return nil, fmt.Errorf("wg show dump failed: %w", err)
	}
 
	var peers []PeerStatus
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
 
	for i, line := range lines {
		if i == 0 {
			continue // skip interface line
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 5 {
			continue
		}
 
		pubkey := fields[0]
		handshakeUnix, err := strconv.ParseInt(fields[4], 10, 64)
		if err != nil {
			continue
		}
 
		var lastHandshake time.Time
		if handshakeUnix > 0 {
			lastHandshake = time.Unix(handshakeUnix, 0)
		}
 
		peers = append(peers, PeerStatus{
			PublicKey:     pubkey,
			LastHandshake: lastHandshake,
		})
	}
 
	return peers, nil
}
 
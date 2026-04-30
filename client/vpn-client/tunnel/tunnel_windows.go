//go:build windows

package tunnel

import (
	"embed"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
)

//go:embed embed/*
var wintunFiles embed.FS

func ensureDll() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	binDir := filepath.Dir(exePath)
	dllPath := filepath.Join(binDir, "wintun.dll")

	// check if its already there
	if _, err := os.Stat(dllPath); os.IsNotExist(err) {
		// now determine which one to grab from the embed/ dir with runtime.goarch
		srcPath := fmt.Sprintf("embed/%s/wintun.dll", runtime.GOARCH)

		dllBytes, err := wintunFiles.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("architecture %s not supported: %w", runtime.GOARCH, err)
		}

		return os.WriteFile(dllPath, dllBytes, 0644)
	}
	return nil
}

func b64ToHex(b64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func Up(privateKey, tunnelIP, serverPubkey, serverEndpoint string) error {
	if err := ensureDll(); err != nil {
		return err
	}

	// Resolve endpoint before passing to wireguard
	host, port, _ := net.SplitHostPort(serverEndpoint)
	ips, _ := net.LookupIP(host)
	if len(ips) == 0 {
		return fmt.Errorf("Could not resolve %s", host)
	}
	// without doing this explicitly wireguard tries to parse the whole "endpoint=domain.duck.dns.org:port"
	resolvedEndpoint := net.JoinHostPort(ips[0].String(), port)

	// Create the Wintun adapter
	// requires Administrator rights (handled by wails.json)
	interfaceName := "VPNClient"
	tunDevice, err := tun.CreateTUN(interfaceName, 1420)
	if err != nil {
		return fmt.Errorf("could not create tun: %w", err)
	}

	// Create the WireGuard device

	logger := device.NewLogger(device.LogLevelSilent, "[VPN] ")
	dev := device.NewDevice(tunDevice, conn.NewDefaultBind(), logger)

	// save this to global so Down() can access it
	activeDevice = dev

	// convert keys from base64 to hex for IpcSet
	privHex, err := b64ToHex(privateKey)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}
	pubHex, err := b64ToHex(serverPubkey)
	if err != nil {
		return fmt.Errorf("invalid server pubkey: %w", err)
	}

	// apply the conf with resolved endpoint  \n doesnt work for UAPI parser sigh
	uapiConf := fmt.Sprintf(`private_key=%s
public_key=%s
endpoint=%s
allowed_ip=0.0.0.0/0
`, privHex, pubHex, resolvedEndpoint)

	err = dev.IpcSet(uapiConf)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	// bring up the interface
	dev.Up()

	ip := strings.SplitN(tunnelIP, "/", 2)[0]
	// Set the IP address on the interface
	setIP := exec.Command("netsh", "interface", "ip", "set", "address",
		"name="+interfaceName, "static", ip, "255.255.255.0")
	if out, err := setIP.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set IP: %s %w", string(out), err)
	}

	return nil
}

var activeDevice *device.Device

func Down() error {
	if activeDevice != nil {
		activeDevice.Close()
		activeDevice = nil
	}
	return nil
}

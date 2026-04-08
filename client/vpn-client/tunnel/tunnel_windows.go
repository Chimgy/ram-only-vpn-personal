//go:build windows

package tunnel

import "errors"

func Up(privateKey, tunnelIP, serverPubkey, serverEndpoint string) error {
	return errors.New("tunnel not implemented on Windows yet")
}

func Down() error {
	return errors.New("tunnel not implemented on Windows yet")
}

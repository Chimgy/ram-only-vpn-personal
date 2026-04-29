package authCrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

// This may seem like overkill and it probably is but:
// Uses TCP Hijacking to snap the connection shut for unauthorized requests.
// This prevents Shodan/Censys from fingerprinting the service, making the port appear "Filtered" or "Dead."
//
// All requests must be pre-encrypted using a key (apiKey) you set. This prevents Man in the middle attacks (MITM)
// bascially, without this the initial connection to 8080 is sniffable. So your apiKey would be accessible (sent via http)
//
// The initial connection is still using TCP so you aren't completely invisible, if you wanted your nodes to be 100% invisible
// you'd need to set up a separate controller/signalling server to gate-keep new connections (check out other repo for this -- but you would need a second pi)

// DeriveKey turns any string (the api key) into a fixed 32 byte key
func DeriveKey(apiKey string) []byte {
	hash := sha256.Sum256([]byte(apiKey))
	return hash[:]
}

func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(rand.Reader, nonce)
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, actualCiphertext, nil)
}

// Package aes provides an AES-256-GCM implementation of encryptor.Encryptor.
package aes

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"github.com/harvor-io/relay/pkg/encryptor"
)

// KeySize is the required length, in bytes, of keys passed to New.
const KeySize = 32 // AES-256

// Encryptor is the AES-256-GCM implementation of encryptor.Encryptor.
// Ciphertexts are the GCM nonce prepended to the sealed output.
type Encryptor struct {
	aead cipher.AEAD
}

var _ encryptor.Encryptor = (*Encryptor)(nil)

// New returns an Encryptor using key for AES-256-GCM. key must be exactly
// KeySize (32) bytes.
func New(key []byte) (*Encryptor, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("aes: key must be %d bytes, got %d", KeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes: new cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aes: new gcm: %w", err)
	}

	return &Encryptor{aead: aead}, nil
}

// Encrypt seals plaintext with a freshly generated nonce, returning the
// nonce prepended to the ciphertext.
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("aes: generate nonce: %w", err)
	}
	return e.aead.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt opens ciphertext produced by Encrypt, which must be at least the
// nonce size.
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("aes: ciphertext shorter than nonce size %d", nonceSize)
	}

	nonce, sealed := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := e.aead.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, fmt.Errorf("aes: decrypt: %w", err)
	}
	return plaintext, nil
}

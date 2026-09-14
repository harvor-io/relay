// Package encryptor defines the boundary for encrypting and decrypting
// secret values at rest. Concrete implementations live in sub-packages.
package encryptor

// Encryptor encrypts and decrypts byte slices.
type Encryptor interface {
	// Encrypt returns the ciphertext for plaintext.
	Encrypt(plaintext []byte) ([]byte, error)

	// Decrypt returns the plaintext for ciphertext.
	Decrypt(ciphertext []byte) ([]byte, error)
}

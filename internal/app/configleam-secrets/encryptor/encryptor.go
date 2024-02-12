package encryptor

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"os"

	chacha20 "golang.org/x/crypto/chacha20poly1305"
)

// Encryptor holds the encryption key and pre-initialized AEAD instance
type Encryptor struct {
	aead cipher.AEAD
}

// NewEncryptor creates a new Encryptor instance with a given key.
// The key must be exactly 32 bytes long for chacha20poly1305.
func NewEncryptor(key string) (*Encryptor, error) {
	if key == "" {
		key = os.Getenv("CG_ENCRYPTION_KEY")
	}

	if key == "" {
		return nil, errors.New("CG_ENCRYPTION_KEY env is not provided")
	}

	if len(key) != chacha20.KeySize {
		return nil, errors.New("key must be 32 bytes long")
	}

	aead, err := chacha20.NewX([]byte(key))
	if err != nil {
		return nil, err
	}
	return &Encryptor{aead: aead}, nil
}

// Encrypt takes a plaintext string and returns an encrypted string.
func (e *Encryptor) Encrypt(_ context.Context, plaintext []byte) ([]byte, error) {
	nonce := make([]byte, chacha20.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := e.aead.Seal(nonce, nonce, []byte(plaintext), nil)

	return ciphertext, nil
}

// Decrypt takes an encrypted string and returns the decrypted plaintext.
func (e *Encryptor) Decrypt(_ context.Context, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < chacha20.NonceSizeX {
		return nil, errors.New("ciphertext too short")
	}

	// Split nonce and ciphertext
	nonce, encryptedMessage := ciphertext[:chacha20.NonceSizeX], ciphertext[chacha20.NonceSizeX:]

	// Decrypt the message using the pre-initialized AEAD instance
	decrypted, err := e.aead.Open(nil, nonce, encryptedMessage, nil)
	if err != nil {
		return nil, err
	}

	return decrypted, nil
}

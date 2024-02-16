package keys

import (
	"context"
	"crypto/rand"
)

const (
	Chars     = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	Length    = 40
	KeyPrefix = "cfg_"
)

type Keys struct{}

func New() *Keys {
	return &Keys{}
}

func (k *Keys) GenerateKey(_ context.Context) (string, error) {
	bytes := make([]byte, Length)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for i, b := range bytes {
		bytes[i] = Chars[b%byte(len(Chars))]
	}

	return KeyPrefix + string(bytes), nil
}

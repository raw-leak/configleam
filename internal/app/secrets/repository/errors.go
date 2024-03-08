package repository

import "fmt"

type SecretNotFoundError struct {
	Key string
}

func (e SecretNotFoundError) Error() string {
	return fmt.Sprintf("secret '%s' not found", e.Key)
}

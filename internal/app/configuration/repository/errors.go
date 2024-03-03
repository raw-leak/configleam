package repository

import "fmt"

type EnvNotFoundError struct {
	Key string
}

func (e EnvNotFoundError) Error() string {
	return fmt.Sprintf("env '%s' not found", e.Key)
}

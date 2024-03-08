package repository

import "fmt"

// EnvNotFoundError is used when an environment is not found.
type EnvNotFoundError struct {
	Key string
}

func (e EnvNotFoundError) Error() string {
	return fmt.Sprintf("env '%s' not found", e.Key)
}

type EnvLockHeldError struct {
	Env string
}

func (e EnvLockHeldError) Error() string {
	return fmt.Sprintf("env '%s' is locked", e.Env)
}

package repository

// EnvParams represents parameters for managing environment metadata.
type EnvParams struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Clone    bool   `json:"clone"`
	Original string `json:"original"`
}

package dto

type CreateAccessKeyParams struct {
	Perms []map[string]string
	Envs  []string
}

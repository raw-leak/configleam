package service

import (
	"context"
	"strings"

	"github.com/raw-leak/configleam/internal/app/secrets/repository"
)

const (
	SecretPhStart = "{{"
	SecretPhEnd   = "}}"
	SecretPattern = "secret."
)

type SecretsService struct {
	repository repository.Repository
}

func New(repository repository.Repository) *SecretsService {
	return &SecretsService{repository: repository}
}

// InsertSecrets insert secrets
func (s *SecretsService) InsertSecrets(ctx context.Context, env string, cfg *map[string]interface{}, populate bool) error {
	for key, value := range *cfg {
		// handle nested maps
		if nestedCfg, ok := value.(map[string]interface{}); ok {
			err := s.InsertSecrets(ctx, env, &nestedCfg, populate)
			if err != nil {
				return err
			}
		} else if arrVal, ok := value.([]interface{}); ok {
			// handle arrays
			err := s.insertSecretsIntoArray(ctx, env, &arrVal, populate)
			if err != nil {
				return err
			}
		} else if strVal, ok := value.(string); ok {
			// handle strings
			updatedVal, err := s.replaceSecretPlaceholders(ctx, env, strVal, populate)
			if err != nil {
				return err
			}

			(*cfg)[key] = updatedVal
		}
		// ignore other types of values (e.g., numbers, booleans)
	}
	return nil
}

// replaceSecretPlaceholders replaces secret placeholders in a string
func (s *SecretsService) replaceSecretPlaceholders(ctx context.Context, env, str string, populate bool) (interface{}, error) {
	if strings.HasPrefix(str, SecretPhStart) && strings.HasSuffix(str, SecretPhEnd) {
		if populate {
			intKey := str[len(SecretPhStart) : len(str)-len(SecretPhEnd)]
			intKey = strings.Trim(intKey, " ")

			if strings.HasPrefix(intKey, SecretPattern) {
				key := intKey[len(SecretPattern):]

				secretValue, err := s.repository.GetSecret(ctx, env, key)
				if err != nil {
					if _, ok := err.(repository.SecretNotFoundError); ok {
						return "", nil
					} else {
						return "", err
					}
				}

				return secretValue, nil
			}
		} else {
			return "", nil
		}

	}

	return str, nil
}

// insertSecretsIntoArray inserts secrets into an array
func (s *SecretsService) insertSecretsIntoArray(ctx context.Context, env string, arr *[]interface{}, populate bool) error {
	for i, val := range *arr {
		switch typedVal := val.(type) {
		case map[string]interface{}:
			err := s.InsertSecrets(ctx, env, &typedVal, populate)
			if err != nil {
				return err
			}
			(*arr)[i] = typedVal
		case []interface{}:
			err := s.insertSecretsIntoArray(ctx, env, &typedVal, populate)
			if err != nil {
				return err
			}
			(*arr)[i] = typedVal
		case string:
			updatedVal, err := s.replaceSecretPlaceholders(ctx, env, typedVal, populate)
			if err != nil {
				return err
			}
			(*arr)[i] = updatedVal
		}

		// ignore other types of array elements
	}
	return nil
}

// UpsertSecrets upserts secrets for environment
func (s SecretsService) UpsertSecrets(ctx context.Context, env string, cfg map[string]interface{}) error {
	err := s.repository.HealthCheck(ctx)
	if err != nil {
		return err
	}

	return s.repository.UpsertSecrets(ctx, env, cfg)
}

// CloneSecrets clones secrets for environment
func (s SecretsService) CloneSecrets(ctx context.Context, cloneEnv, newEnv string) error {
	err := s.repository.HealthCheck(ctx)
	if err != nil {
		return err
	}

	return s.repository.CloneSecrets(ctx, cloneEnv, newEnv)
}

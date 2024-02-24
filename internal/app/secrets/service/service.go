package service

import (
	"context"
	"strings"
)

const (
	SecretPhStart = "{{"
	SecretPhEnd   = "}}"
	SecretPattern = "secret."
)

type Repository interface {
	GetSecret(ctx context.Context, env string, key string) (interface{}, error)
	UpsertSecrets(ctx context.Context, env string, secrets map[string]interface{}) error
	HealthCheck(ctx context.Context) error
}

type SecretsService struct {
	repository Repository
}

func New(repository Repository) *SecretsService {
	return &SecretsService{repository: repository}
}

// Method to insert secrets into a map[string]interface{}
func (s *SecretsService) InsertSecrets(ctx context.Context, env string, cfg *map[string]interface{}) error {
	for key, value := range *cfg {
		// handle nested maps
		if nestedCfg, ok := value.(map[string]interface{}); ok {
			err := s.InsertSecrets(ctx, env, &nestedCfg)
			if err != nil {
				return err
			}
		} else if arrVal, ok := value.([]interface{}); ok {
			// handle arrays
			err := s.insertSecretsIntoArray(ctx, env, &arrVal)
			if err != nil {
				return err
			}
		} else if strVal, ok := value.(string); ok {
			// handle strings
			updatedVal, err := s.replaceSecretPlaceholders(ctx, env, strVal)
			if err != nil {
				return err
			}

			(*cfg)[key] = updatedVal
		}
		// ignore other types of values (e.g., numbers, booleans)
	}
	return nil
}

// Method to replace secret placeholders in a string
func (s *SecretsService) replaceSecretPlaceholders(ctx context.Context, env, str string) (interface{}, error) {
	if strings.HasPrefix(str, SecretPhStart) && strings.HasSuffix(str, SecretPhEnd) {
		intKey := str[len(SecretPhStart) : len(str)-len(SecretPhEnd)]
		intKey = strings.Trim(intKey, " ")

		if strings.HasPrefix(intKey, SecretPattern) {
			key := intKey[len(SecretPattern):]

			secretValue, err := s.repository.GetSecret(ctx, env, key)
			if err != nil {
				return "", err
			}

			return secretValue, nil
		}

	}

	return str, nil
}

// Method to insert secrets into an array of interface{}
func (s *SecretsService) insertSecretsIntoArray(ctx context.Context, env string, arr *[]interface{}) error {
	for i, val := range *arr {
		switch typedVal := val.(type) {
		case map[string]interface{}:
			err := s.InsertSecrets(ctx, env, &typedVal)
			if err != nil {
				return err
			}
			(*arr)[i] = typedVal
		case []interface{}:
			err := s.insertSecretsIntoArray(ctx, env, &typedVal)
			if err != nil {
				return err
			}
			(*arr)[i] = typedVal
		case string:
			updatedVal, err := s.replaceSecretPlaceholders(ctx, env, typedVal)
			if err != nil {
				return err
			}
			(*arr)[i] = updatedVal
		}

		// ignore other types of array elements
	}
	return nil
}

// Method to upsert secrets for environment
func (s *SecretsService) UpsertSecrets(ctx context.Context, env string, cfg map[string]interface{}) error {
	err := s.repository.HealthCheck(ctx)
	if err != nil {
		return err
	}

	return s.repository.UpsertSecrets(ctx, env, cfg)
}

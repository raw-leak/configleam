package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/raw-leak/configleam/internal/app/configleam-secrets/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
	mock.Mock
}

// GetSecret is a mock implementation of the GetSecret method
func (m *MockRepository) GetSecret(ctx context.Context, env string, key string) (interface{}, error) {
	args := m.Called(ctx, env, key)
	return args.String(0), args.Error(1)
}

// UpsertSecrets is a mock implementation of the UpsertSecrets method
func (m *MockRepository) UpsertSecrets(ctx context.Context, env string, secrets map[string]interface{}) error {
	args := m.Called(ctx, env, secrets)
	return args.Error(1)
}

// HealthCheck is a mock implementation of the HealthCheck method
func (m *MockRepository) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(1)
}

type ConfigleamSecretsSuite struct {
	suite.Suite
	secrets    service.ConfigleamSecrets
	repository *MockRepository
}

func TestConfigleamSecretsSuite(t *testing.T) {
	suite.Run(t, new(ConfigleamSecretsSuite))
}

func (suite *ConfigleamSecretsSuite) SetupSuite() {
	suite.repository = &MockRepository{}
	suite.secrets = *service.New(suite.repository)
}

func (suite *ConfigleamSecretsSuite) TestInsertSecrets() {
	t := suite.T()

	type GetSecretMock struct {
		key   string
		value string
		err   error
	}

	testCases := []struct {
		name          string
		env           string
		inputCfg      map[string]interface{}
		expectedCfg   map[string]interface{}
		expectedErr   error
		GetSecretMock []GetSecretMock
	}{
		{
			name: "Replace well defined secret in a map",
			env:  "develop",
			GetSecretMock: []GetSecretMock{
				{key: "test", value: "test-value"},
			},
			inputCfg: map[string]interface{}{
				"some-key": "{{secret.test}}",
			},
			expectedCfg: map[string]interface{}{
				"some-key": "test-value",
			},
			expectedErr: nil,
		},
		{
			name: "Replace with spaces on the left secret in a map",
			env:  "develop",
			GetSecretMock: []GetSecretMock{
				{key: "test", value: "test-value"},
			},
			inputCfg: map[string]interface{}{
				"some-key": "{{ secret.test}}",
			},
			expectedCfg: map[string]interface{}{
				"some-key": "test-value",
			},
			expectedErr: nil,
		},
		{
			name: "Replace with spaces on the right secret in a map",
			env:  "develop",
			GetSecretMock: []GetSecretMock{
				{key: "test", value: "test-value"},
			},
			inputCfg: map[string]interface{}{
				"some-key": "{{secret.test }}",
			},
			expectedCfg: map[string]interface{}{
				"some-key": "test-value",
			},
			expectedErr: nil,
		},
		{
			name: "Replace with spaces on the right and left secret in a map",
			env:  "develop",
			GetSecretMock: []GetSecretMock{
				{key: "test", value: "test-value"},
			},
			inputCfg: map[string]interface{}{
				"some-key": "{{ secret.test }}",
			},
			expectedCfg: map[string]interface{}{
				"some-key": "test-value",
			},
			expectedErr: nil,
		},
		{
			name: "Replace multiple secret in a map where secrets are simple strings",
			env:  "develop",
			GetSecretMock: []GetSecretMock{
				{key: "test1", value: "test-value-1"},
				{key: "test2", value: "test-value-2"},
				{key: "test3", value: "test-value-3"},
			},
			inputCfg: map[string]interface{}{
				"some-key-1": "{{secret.test1}}",
				"some-key-2": "{{secret.test2}}",
				"some-key-3": "{{secret.test3}}",
			},
			expectedCfg: map[string]interface{}{
				"some-key-1": "test-value-1",
				"some-key-2": "test-value-2",
				"some-key-3": "test-value-3",
			},
			expectedErr: nil,
		},
		{
			name: "Simple map with nested secrets",
			env:  "develop",
			GetSecretMock: []GetSecretMock{
				{key: "test1", value: "test-value-1"},
				{key: "test2", value: "test-value-2"},
			},
			inputCfg: map[string]interface{}{
				"nested-map": map[string]interface{}{
					"key1": "{{secret.test1}}",
					"key2": "{{secret.test2}}",
				},
			},
			expectedCfg: map[string]interface{}{
				"nested-map": map[string]interface{}{
					"key1": "test-value-1",
					"key2": "test-value-2",
				},
			},
			expectedErr: nil,
		},
		{
			name: "Simple map with array of secrets",
			env:  "develop",
			GetSecretMock: []GetSecretMock{
				{key: "test1", value: "test-value-1"},
				{key: "test2", value: "test-value-2"},
			},
			inputCfg: map[string]interface{}{
				"secret-array": []interface{}{
					"{{secret.test1}}",
					"{{secret.test2}}",
				},
			},
			expectedCfg: map[string]interface{}{
				"secret-array": []interface{}{
					"test-value-1",
					"test-value-2",
				},
			},
			expectedErr: nil,
		}, {
			name: "Invalid secret placeholder format (end)",
			env:  "develop",
			inputCfg: map[string]interface{}{
				"invalid-secret": "{{secret.invalid-secret}",
			},
			expectedCfg: map[string]interface{}{
				"invalid-secret": "{{secret.invalid-secret}",
			},
			expectedErr: nil,
		},
		{
			name: "Invalid secret placeholder format (init)",
			env:  "develop",
			inputCfg: map[string]interface{}{
				"invalid-secret": "{secret.invalid-secret}}",
			},
			expectedCfg: map[string]interface{}{
				"invalid-secret": "{secret.invalid-secret}}",
			},
			expectedErr: nil,
		},
		{
			name: "Input config without secrets",
			env:  "develop",
			inputCfg: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			expectedCfg: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			expectedErr: nil,
		},
		{
			name: "Non-string value",
			env:  "develop",
			inputCfg: map[string]interface{}{
				"int-value":  42,
				"bool-value": true,
			},
			expectedCfg: map[string]interface{}{
				"int-value":  42,
				"bool-value": true,
			},
			expectedErr: nil,
		},
		{
			name: "Complex scenario with nested structures",
			env:  "develop",
			inputCfg: map[string]interface{}{
				"key1": "value1",
				"key2": map[string]interface{}{
					"nested-key1": "{{secret.test1}}",
					"nested-key2": 42,
					"nested-key3": true,
					"nested-key4": []interface{}{
						map[string]interface{}{
							"array-key1": "{{secret.test2}}",
							"array-key2": 123,
						},
						map[string]interface{}{
							"array-key3": "{{secret.test3}}",
							"array-key4": false,
						},
					},
				},
				"key3": []interface{}{
					"array-value1",
					"{{secret.test4}}",
					map[string]interface{}{
						"nested-array-key1": "{{secret.test5}}",
						"nested-array-key2": 456,
					},
				},
			},
			expectedCfg: map[string]interface{}{
				"key1": "value1",
				"key2": map[string]interface{}{
					"nested-key1": "test-value-1",
					"nested-key2": 42,
					"nested-key3": true,
					"nested-key4": []interface{}{
						map[string]interface{}{
							"array-key1": "test-value-2",
							"array-key2": 123,
						},
						map[string]interface{}{
							"array-key3": "test-value-3",
							"array-key4": false,
						},
					},
				},
				"key3": []interface{}{
					"array-value1",
					"test-value-4",
					map[string]interface{}{
						"nested-array-key1": "test-value-5",
						"nested-array-key2": 456,
					},
				},
			},
			GetSecretMock: []GetSecretMock{
				{key: "test1", value: "test-value-1"},
				{key: "test2", value: "test-value-2"},
				{key: "test3", value: "test-value-3"},
				{key: "test4", value: "test-value-4"},
				{key: "test5", value: "test-value-5"},
			},
			expectedErr: nil,
		},
		{
			name: "Very nested complex scenario",
			env:  "develop",
			inputCfg: map[string]interface{}{
				"key1": "value1",
				"key2": map[string]interface{}{
					"nested-key1": "{{secret.test1}}",
					"nested-key2": 42,
					"nested-key3": true,
					"nested-key4": []interface{}{
						map[string]interface{}{
							"array-key1": "{{secret.test2}}",
							"array-key2": 123,
						},
						map[string]interface{}{
							"array-key3": "{{secret.test3}}",
							"array-key4": false,
							"array-key5": []interface{}{
								map[string]interface{}{
									"inner-array-key1": "{{secret.test4}}",
									"inner-array-key2": 456,
								},
								map[string]interface{}{
									"inner-array-key3": "{{secret.test5}}",
									"inner-array-key4": "string-value",
								},
							},
						},
					},
				},
				"key3": []interface{}{
					"array-value1",
					"{{secret.test6}}",
					map[string]interface{}{
						"nested-array-key1": "{{secret.test7}}",
						"nested-array-key2": 789,
						"nested-array-key3": []interface{}{
							"inner-array-value1",
							"{{secret.test8}}",
						},
					},
				},
			},
			expectedCfg: map[string]interface{}{
				"key1": "value1",
				"key2": map[string]interface{}{
					"nested-key1": "test-value-1",
					"nested-key2": 42,
					"nested-key3": true,
					"nested-key4": []interface{}{
						map[string]interface{}{
							"array-key1": "test-value-2",
							"array-key2": 123,
						},
						map[string]interface{}{
							"array-key3": "test-value-3",
							"array-key4": false,
							"array-key5": []interface{}{
								map[string]interface{}{
									"inner-array-key1": "test-value-4",
									"inner-array-key2": 456,
								},
								map[string]interface{}{
									"inner-array-key3": "test-value-5",
									"inner-array-key4": "string-value",
								},
							},
						},
					},
				},
				"key3": []interface{}{
					"array-value1",
					"test-value-6",
					map[string]interface{}{
						"nested-array-key1": "test-value-7",
						"nested-array-key2": 789,
						"nested-array-key3": []interface{}{
							"inner-array-value1",
							"test-value-8",
						},
					},
				},
			},
			GetSecretMock: []GetSecretMock{
				{key: "test1", value: "test-value-1"},
				{key: "test2", value: "test-value-2"},
				{key: "test3", value: "test-value-3"},
				{key: "test4", value: "test-value-4"},
				{key: "test5", value: "test-value-5"},
				{key: "test6", value: "test-value-6"},
				{key: "test7", value: "test-value-7"},
				{key: "test8", value: "test-value-8"},
			},
			expectedErr: nil,
		},
		{
			name: "Single secrets replacement with one error",
			env:  "develop",
			GetSecretMock: []GetSecretMock{
				{key: "test", value: "", err: errors.New("some error")},
			},
			inputCfg: map[string]interface{}{
				"some-key": "{{ secret.test }}",
			},
			expectedCfg: map[string]interface{}{
				"some-key": "test-value",
			},
			expectedErr: errors.New("some error"),
		},
		{
			name: "Multiple secrets replacement with one error",
			env:  "develop",
			GetSecretMock: []GetSecretMock{
				{key: "test1", value: "test-value-1", err: nil},
				{key: "test2", value: "", err: errors.New("error retrieving test2")},
				{key: "test3", value: "test-value-3", err: nil},
			},
			inputCfg: map[string]interface{}{
				"some-key-1": "{{secret.test1}}",
				"some-key-2": "{{secret.test2}}",
				"some-key-3": "{{secret.test3}}",
			},
			expectedCfg: map[string]interface{}{
				"some-key-1": "test-value-1",
				"some-key-2": "{{secret.test2}}",
				"some-key-3": "test-value-3",
			},
			expectedErr: errors.New("error retrieving test2"),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ctx := context.Background()

			for _, mock := range tc.GetSecretMock {
				suite.repository.On("GetSecret", ctx, tc.env, mock.key).Return(mock.value, mock.err).Once()
			}

			err := suite.secrets.InsertSecrets(ctx, tc.env, &tc.inputCfg)

			// Assert error, if expected
			if tc.expectedErr != nil {
				suite.Assert().EqualError(err, tc.expectedErr.Error())
			} else {
				// Assert no error
				suite.Assert().NoError(err)

				// Assert that inputCfg has been updated as expected
				assert.Equal(t, tc.expectedCfg, tc.inputCfg)

				// Assert that all expected calls to GetSecret were made
				suite.repository.AssertExpectations(t)
			}
		})
	}
}

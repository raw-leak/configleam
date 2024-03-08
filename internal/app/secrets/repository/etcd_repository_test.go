package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/raw-leak/configleam/internal/app/secrets/repository"
	"github.com/raw-leak/configleam/internal/pkg/encryptor"
	"github.com/raw-leak/configleam/internal/pkg/etcd"
	"github.com/stretchr/testify/suite"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdSecretsRepositorySuite struct {
	suite.Suite

	repository *repository.EtcdRepository
	client     *clientv3.Client

	encryptor *encryptor.Encryptor
	key       string
}

func TestEtcdSecretsRepositorySuite(t *testing.T) {
	suite.Run(t, new(EtcdSecretsRepositorySuite))
}

func (suite *EtcdSecretsRepositorySuite) SetupSuite() {

	addrs := "http://localhost:8079"

	client, err := clientv3.New(clientv3.Config{
		Endpoints: []string{addrs},
	})
	suite.NoErrorf(err, "error connecting to etcd server %s", addrs)

	suite.client = client

	suite.key = "01234567890123456789012345678901"
	suite.encryptor, err = encryptor.NewEncryptor(suite.key)
	suite.NoError(err)

	suite.repository = repository.NewEtcdRepository(&etcd.Etcd{Client: suite.client}, suite.encryptor)
}

func (suite *EtcdSecretsRepositorySuite) TearDownSuite() {
	suite.client.Close()
}

func (suite *EtcdSecretsRepositorySuite) BeforeTest(testName string) {
	_, err := suite.client.Delete(context.Background(), "", clientv3.WithPrefix())
	suite.NoErrorf(err, "Deleting all data from ETCD before each test within the test: %s", testName)
}

func (suite *EtcdSecretsRepositorySuite) TestGetSecret() {
	type prePopulateData struct {
		key   string
		value interface{}
	}

	testCases := []struct {
		name          string
		env           string
		fullKey       string
		prePopulate   []prePopulateData
		expectedValue interface{}
		expectedError error
	}{
		{
			name:    "Valid plain secret key with simple map value",
			env:     "development",
			fullKey: "some-key-1",
			prePopulate: []prePopulateData{
				{key: "some-key-1", value: map[string]interface{}{"key1": "value1"}},
				{key: "some-key-2", value: map[string]interface{}{"key2": "value2"}},
			},
			expectedValue: map[string]interface{}{"key1": "value1"},
			expectedError: nil,
		},
		{
			name:    "Valid plain secret key with deeply nested map value",
			env:     "development",
			fullKey: "some-key-1",
			prePopulate: []prePopulateData{
				{key: "some-key-1", value: map[string]interface{}{"nested": map[string]interface{}{"key1": "value1"}}},
			},
			expectedValue: map[string]interface{}{"nested": map[string]interface{}{"key1": "value1"}},
			expectedError: nil,
		},
		{
			name:    "Valid plain secret key with array value",
			env:     "development",
			fullKey: "some-key-1",
			prePopulate: []prePopulateData{
				{key: "some-key-1", value: []interface{}{"value1", "value2", "value3"}},
			},
			expectedValue: []interface{}{"value1", "value2", "value3"},
			expectedError: nil,
		},
		{
			name:    "Valid nested secret key with nested map value with multiple values",
			env:     "development",
			fullKey: "database.password",
			prePopulate: []prePopulateData{
				{key: "database", value: map[string]interface{}{
					"password": "db_password",
					"one": map[string]interface{}{
						"password": "db_one_password",
					},
					"two": map[string]interface{}{
						"password": "db_one_password",
					},
				}},
			},
			expectedValue: "db_password",
			expectedError: nil,
		},
		{
			name:    "Valid deeply nested secret key with nested map value with multiple values",
			env:     "development",
			fullKey: "database.one.password",
			prePopulate: []prePopulateData{
				{key: "database", value: map[string]interface{}{
					"password": "db_password",
					"one": map[string]interface{}{
						"password": "db_one_password",
					},
					"two": map[string]interface{}{
						"password": "db_one_password",
					},
				}},
			},
			expectedValue: "db_one_password",
			expectedError: nil,
		},
		{
			name:    "Valid very deeply nested secret key with nested map value with multiple values",
			env:     "development",
			fullKey: "database.two.three.password",
			prePopulate: []prePopulateData{
				{key: "database", value: map[string]interface{}{
					"password": "db_password",
					"one": map[string]interface{}{
						"password": "db_one_password",
					},
					"two": map[string]interface{}{
						"three": map[string]interface{}{
							"password": "db_three_password",
						},
					},
				}},
			},
			expectedValue: "db_three_password",
			expectedError: nil,
		},
		{
			name:    "Not valid very deeply nested unexisting secret key with nested map value with multiple values",
			env:     "development",
			fullKey: "database.password.password",
			prePopulate: []prePopulateData{
				{key: "database", value: map[string]interface{}{
					"password": "db_password",
					"one": map[string]interface{}{
						"password": "db_one_password",
					},
					"two": map[string]interface{}{
						"three": map[string]interface{}{
							"password": "db_three_password",
						},
					},
				}},
			},
			expectedValue: nil,
			expectedError: repository.SecretNotFoundError{Key: "database.password.password"},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.BeforeTest(tc.name)
			ctx := context.Background()

			for _, data := range tc.prePopulate {
				value, err := json.Marshal(data.value)
				suite.NoError(err)

				encrypted, err := suite.encryptor.Encrypt(ctx, value)
				suite.NoError(err)

				_, err = suite.client.Put(ctx, suite.repository.GetSecretKey(tc.env, data.key), string(encrypted))
				suite.NoError(err)
			}

			res, err := suite.client.Get(ctx, repository.SecretPrefix, clientv3.WithPrefix())
			suite.NoError(err)
			keys := res.Kvs

			suite.NoError(err)
			suite.Equal(len(keys), len(tc.prePopulate))

			value, err := suite.repository.GetSecret(ctx, tc.env, tc.fullKey)

			if tc.expectedError != nil {
				suite.Equal(tc.expectedError, err)
			} else {
				suite.NoError(err)
				suite.Equal(tc.expectedValue, value)
			}

		})
	}
}

func (suite *EtcdSecretsRepositorySuite) TestUpsertSecrets() {
	type prePopulateData struct {
		key   string
		value interface{}
	}

	testCases := []struct {
		name        string
		prePopulate []prePopulateData
		inputValue  map[string]interface{}
		inputEnv    string
		expectedErr error
	}{
		{
			name:        "Upserting simple new key",
			inputEnv:    "develop",
			inputValue:  map[string]interface{}{"key": "value"},
			prePopulate: []prePopulateData{},
			expectedErr: nil,
		},
		{
			name:     "Upserting nested map with single key",
			inputEnv: "develop",
			inputValue: map[string]interface{}{
				"key": map[string]interface{}{
					"key1": "value1",
				},
			},
			prePopulate: []prePopulateData{},
			expectedErr: nil,
		},
		{
			name:     "Upserting new key with deeply nested map",
			inputEnv: "develop",
			inputValue: map[string]interface{}{
				"key": map[string]interface{}{
					"key1": map[string]interface{}{
						"key2": "value2",
					},
				},
			},
			prePopulate: []prePopulateData{},
			expectedErr: nil,
		},
		{
			name:     "Upserting nested map with existing key",
			inputEnv: "develop",
			inputValue: map[string]interface{}{
				"key": map[string]interface{}{
					"key1": "updated_value1",
				},
			},
			prePopulate: []prePopulateData{
				{key: "key", value: map[string]interface{}{
					"key1": "value1",
				}},
			},
			expectedErr: nil,
		},
		{
			name:     "Upserting deeply nested map with existing key",
			inputEnv: "develop",
			inputValue: map[string]interface{}{
				"key": map[string]interface{}{
					"key1": map[string]interface{}{
						"key2": "updated_value2",
					},
				},
			},
			prePopulate: []prePopulateData{
				{key: "key", value: map[string]interface{}{
					"key1": map[string]interface{}{
						"key2": "value2",
					},
				}},
			},
			expectedErr: nil,
		},
		{
			name:        "Upserting new key with array value",
			inputEnv:    "develop",
			inputValue:  map[string]interface{}{"key": []interface{}{"value1", "value2"}},
			prePopulate: []prePopulateData{},
			expectedErr: nil,
		},
		{
			name:     "Upserting new key with existing array value",
			inputEnv: "develop",
			inputValue: map[string]interface{}{
				"key": []interface{}{"updated_value1", "value2", "value3"},
			},
			prePopulate: []prePopulateData{
				{key: "key", value: []interface{}{"value1", "value2", "value3"}},
			},
			expectedErr: nil,
		},
		{
			name:        "Upserting new key with existing non-map value",
			inputEnv:    "develop",
			inputValue:  map[string]interface{}{"key": "updated_value"},
			prePopulate: []prePopulateData{{key: "key", value: "value"}},
			expectedErr: nil,
		},
		{
			name:        "Upserting new key with nil value",
			inputEnv:    "develop",
			inputValue:  map[string]interface{}{"key": nil},
			prePopulate: []prePopulateData{},
			expectedErr: errors.New("'nil' can not be used as value for the key 'key'"),
		},
		{
			name:        "Upserting new key with invalid key path",
			inputEnv:    "develop",
			inputValue:  map[string]interface{}{"invalid.key": "value"},
			prePopulate: []prePopulateData{},
			expectedErr: fmt.Errorf("the secret configuration key 'invalid.key' is malformed"),
		},
		{
			name:     "Upserting multiple new keys",
			inputEnv: "develop",
			inputValue: map[string]interface{}{
				"key1": "value1",
				"key2": map[string]interface{}{
					"nestedKey": "nestedValue",
				},
				"key3": []interface{}{"value3", "value4"},
			},
			prePopulate: []prePopulateData{},
			expectedErr: nil,
		},
		{
			name:     "Upserting multiple keys with existing values",
			inputEnv: "develop",
			inputValue: map[string]interface{}{
				"key1": "updated_value1",
				"key2": map[string]interface{}{
					"nestedKey": "updated_nestedValue",
				},
				"key3": []interface{}{"updated_value3", "updated_value4"},
			},
			prePopulate: []prePopulateData{
				{key: "key1", value: "value1"},
				{key: "key2", value: map[string]interface{}{
					"nestedKey": "nestedValue",
				}},
				{key: "key3", value: []interface{}{"value3", "value4"}},
			},
			expectedErr: nil,
		},
		{
			name:     "Upserting multiple new keys with one key having invalid path",
			inputEnv: "develop",
			inputValue: map[string]interface{}{
				"valid":       "value",
				"invalid.key": "value",
			},
			prePopulate: []prePopulateData{},
			expectedErr: fmt.Errorf("the secret configuration key 'invalid.key' is malformed"),
		},
		{
			name:     "Upserting multiple new keys with nil value",
			inputEnv: "develop",
			inputValue: map[string]interface{}{
				"key1": "value1",
				"key2": nil,
				"key3": "value3",
			},
			prePopulate: []prePopulateData{},
			expectedErr: errors.New("'nil' can not be used as value for the key 'key2'"),
		},
		{
			name:        "Upserting multiple new keys with empty configuration",
			inputEnv:    "develop",
			inputValue:  map[string]interface{}{},
			prePopulate: []prePopulateData{},
			expectedErr: errors.New("provided configuration is empty"),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {

			suite.BeforeTest(tc.name)
			ctx := context.Background()

			for _, data := range tc.prePopulate {
				value, err := json.Marshal(data.value)
				suite.NoError(err)

				encrypted, err := suite.encryptor.Encrypt(ctx, value)
				suite.NoError(err)

				_, err = suite.client.Put(ctx, suite.repository.GetSecretKey(tc.inputEnv, data.key), string(encrypted))
				suite.NoError(err)
			}

			res, err := suite.client.Get(ctx, repository.SecretPrefix, clientv3.WithPrefix())
			suite.NoError(err)

			keys := res.Kvs
			suite.Equal(len(keys), len(tc.prePopulate))

			err = suite.repository.UpsertSecrets(ctx, tc.inputEnv, tc.inputValue)

			if tc.expectedErr != nil {
				suite.Equal(tc.expectedErr, err)

				for _, pre := range tc.prePopulate {
					value, err := suite.repository.GetSecret(context.Background(), tc.inputEnv, pre.key)
					suite.NoError(err)
					suite.Equal(value, pre.value)
				}
			} else {
				suite.NoError(err)

				// Verify that the inserted secrets match the expected values
				for key, value := range tc.inputValue {
					expectedValue := value
					if expectedValue == nil {
						expectedValue = "nil" // Handle nil value
					}

					// Get the stored secret from Redis and decrypt it
					res, err := suite.client.Get(ctx, suite.repository.GetSecretKey(tc.inputEnv, key))
					suite.NoError(err)
					encrypted := res.Kvs[0].Value

					decrypted, err := suite.encryptor.Decrypt(ctx, encrypted)
					suite.NoError(err)

					// Unmarshal the decrypted secret to compare with the expected value
					var storedValue interface{}
					err = json.Unmarshal([]byte(decrypted), &storedValue)
					suite.NoError(err)

					suite.Equal(expectedValue, storedValue)
				}
			}
		})
	}
}

func (suite *EtcdSecretsRepositorySuite) TestCloneSecrets() {
	testCases := []struct {
		name        string
		env         string
		newEnv      string
		prePopulate map[string]interface{}
		expectedErr error
	}{
		{
			name:        "Cloning secrets for an environment with one simple secret",
			env:         "dev",
			newEnv:      "clone",
			prePopulate: map[string]interface{}{"key1": "value1"},
			expectedErr: nil,
		},
		{
			name:   "Cloning secrets for an environment with multiple nested secret",
			env:    "dev",
			newEnv: "clone",
			prePopulate: map[string]interface{}{
				"key1": map[string]interface{}{
					"nestedKey1": true,
				},
				"key2": map[string]interface{}{
					"nestedKey2": true,
				},
				"key3": []interface{}{"one", "two", "three"},
			},
			expectedErr: nil,
		},
		{
			name:        "Cloning secrets when the source environment is empty",
			env:         "empty",
			newEnv:      "clone",
			prePopulate: nil,
			expectedErr: nil,
		},
		{
			name:        "Cloning secrets to an environment that already exists",
			env:         "existing",
			newEnv:      "clone",
			prePopulate: map[string]interface{}{"key1": "value1"},
			expectedErr: nil,
		},
		{
			name:        "Cloning secrets with keys containing special characters",
			env:         "special_chars",
			newEnv:      "clone",
			prePopulate: map[string]interface{}{"key with spaces": "value1", "key/with/slashes": "value2"},
			expectedErr: nil,
		},
		// Add more test cases as needed

	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Arrange
			suite.BeforeTest(tc.name)
			ctx := context.Background()

			if tc.prePopulate != nil {
				err := suite.repository.UpsertSecrets(ctx, tc.env, tc.prePopulate)
				suite.NoError(err)
			}

			res, err := suite.client.Get(ctx, suite.repository.GetBaseKey(""), clientv3.WithPrefix())
			suite.NoError(err)

			keys := res.Kvs
			suite.Equal(len(keys), len(tc.prePopulate))

			// Act
			err = suite.repository.CloneSecrets(ctx, tc.env, tc.newEnv)

			// Assert
			if tc.expectedErr != nil {
				suite.Equal(tc.expectedErr, err)

				res, err := suite.client.Get(ctx, suite.repository.GetBaseKey(""), clientv3.WithPrefix())
				suite.NoError(err)

				keys := res.Kvs
				suite.Equal(len(keys), len(tc.prePopulate))

				// ensure the original value has not been changed
				for key, expectedValue := range tc.prePopulate {
					res, err := suite.client.Get(ctx, suite.repository.GetSecretKey(tc.env, key))
					suite.NoError(err)

					decrypted, err := suite.encryptor.Decrypt(ctx, res.Kvs[0].Value)
					suite.NoError(err)

					var actualValue interface{}
					err = json.Unmarshal([]byte(decrypted), &actualValue)
					suite.NoError(err)

					suite.Equal(expectedValue, actualValue)
				}
			} else {
				suite.NoError(err)

				// ensure the amount of original + cloned env keys is the expected
				res, err := suite.client.Get(ctx, suite.repository.GetBaseKey(""), clientv3.WithPrefix())
				suite.NoError(err)

				keys := res.Kvs
				suite.Equal(len(keys), len(tc.prePopulate)*2)

				for key, expectedValue := range tc.prePopulate {
					// ensure the original value has not been changed
					res, err := suite.client.Get(ctx, suite.repository.GetSecretKey(tc.env, key))
					suite.NoError(err)

					decrypted, err := suite.encryptor.Decrypt(ctx, res.Kvs[0].Value)
					suite.NoError(err)

					var actualValue interface{}
					err = json.Unmarshal([]byte(decrypted), &actualValue)
					suite.NoError(err)

					suite.Equal(expectedValue, actualValue)

					// ensure the cloned value has been created with the same value
					res, err = suite.client.Get(ctx, suite.repository.GetSecretKey(tc.newEnv, key))
					suite.NoError(err)

					decrypted, err = suite.encryptor.Decrypt(ctx, res.Kvs[0].Value)
					suite.NoError(err)

					var clonedValue interface{}
					err = json.Unmarshal([]byte(decrypted), &clonedValue)
					suite.NoError(err)

					suite.Equal(expectedValue, clonedValue)
				}
			}
		})
	}
}

func (suite *EtcdSecretsRepositorySuite) TestDeleteSecrets() {
	testCases := []struct {
		name        string
		env         string
		prePopulate map[string]interface{}
		expectedErr error
	}{
		{
			name: "Deleting secrets for an environment with multiple nested secret",
			env:  "dev",
			prePopulate: map[string]interface{}{
				"key1": map[string]interface{}{
					"nestedKey1": true,
				},
				"key2": map[string]interface{}{
					"nestedKey2": true,
				},
				"key3": []interface{}{"one", "two", "three"},
			},
			expectedErr: nil,
		},
		{
			name:        "Deleting unexisting secrets",
			env:         "dev",
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Arrange
			suite.BeforeTest(tc.name)
			ctx := context.Background()

			if tc.prePopulate != nil {
				err := suite.repository.UpsertSecrets(ctx, tc.env, tc.prePopulate)
				suite.NoError(err)
			}

			res, err := suite.client.Get(ctx, repository.SecretPrefix, clientv3.WithPrefix())
			suite.NoError(err)
			keys := res.Kvs

			suite.Equal(len(keys), len(tc.prePopulate))

			// Act
			err = suite.repository.DeleteSecrets(ctx, tc.env)

			// Assert
			if tc.expectedErr != nil {
				suite.Equal(tc.expectedErr, err)

				res, err := suite.client.Get(ctx, repository.SecretPrefix, clientv3.WithPrefix())
				suite.NoError(err)
				keys := res.Kvs

				suite.Equal(len(keys), len(tc.prePopulate))

				// ensure the original value has not been changed
				for key, expectedValue := range tc.prePopulate {
					res, err := suite.client.Get(ctx, suite.repository.GetSecretKey(tc.env, key))
					suite.NoError(err)

					decrypted, err := suite.encryptor.Decrypt(ctx, res.Kvs[0].Value)
					suite.NoError(err)

					var actualValue interface{}
					err = json.Unmarshal([]byte(decrypted), &actualValue)
					suite.NoError(err)

					suite.Equal(expectedValue, actualValue)
				}
			} else {
				suite.NoError(err)

				res, err := suite.client.Get(ctx, repository.SecretPrefix, clientv3.WithPrefix())
				suite.NoError(err)
				keys := res.Kvs

				suite.Equal(len(keys), 0)
			}
		})
	}
}

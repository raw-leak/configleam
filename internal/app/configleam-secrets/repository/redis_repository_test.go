package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/raw-leak/configleam/internal/app/configleam-secrets/repository"
	"github.com/raw-leak/configleam/internal/pkg/encryptor"
	rds "github.com/raw-leak/configleam/internal/pkg/redis"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
)

type RedisConfigleamSecretsRepositorySuite struct {
	suite.Suite

	repository *repository.RedisRepository
	client     *redis.Client

	encryptor *encryptor.Encryptor
	key       string
}

func TestRedisConfigleamSecretsRepositorySuite(t *testing.T) {
	suite.Run(t, new(RedisConfigleamSecretsRepositorySuite))
}

func (suite *RedisConfigleamSecretsRepositorySuite) SetupSuite() {
	var err error

	suite.client = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	suite.key = "01234567890123456789012345678901"
	suite.encryptor, err = encryptor.NewEncryptor(suite.key)
	suite.Require().NoError(err)

	suite.repository = repository.NewRedisRepository(&rds.Redis{Client: suite.client}, suite.encryptor)
}

func (suite *RedisConfigleamSecretsRepositorySuite) TearDownSuite() {
	suite.client.Close()
}

func (suite *RedisConfigleamSecretsRepositorySuite) BeforeTest(testName string) {
	err := suite.client.FlushAll(context.Background()).Err()
	suite.Require().NoErrorf(err, "Flushing all data from redis before each test within the test: %s", testName)
}

func (suite *RedisConfigleamSecretsRepositorySuite) TestGetSecret() {
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
			expectedError: errors.New("not found value for secret 'database.password.password'"),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.BeforeTest(tc.name)
			ctx := context.Background()

			// Pre-populate Redis with test data
			for _, data := range tc.prePopulate {
				value, err := json.Marshal(data.value)
				suite.Require().NoError(err)

				encrypted, err := suite.encryptor.Encrypt(ctx, value)
				suite.Require().NoError(err)

				err = suite.client.Set(ctx, suite.repository.GetSecretKey(tc.env, data.key), encrypted, 0).Err()
				suite.Require().NoError(err)

			}

			keys, err := suite.client.Keys(ctx, "*").Result()
			suite.Require().NoError(err)
			suite.Require().Equal(len(keys), len(tc.prePopulate))

			value, err := suite.repository.GetSecret(ctx, tc.env, tc.fullKey)

			if tc.expectedError != nil {
				suite.Assert().Equal(tc.expectedError, err)
			} else {
				suite.Assert().NoError(err)
				suite.Assert().Equal(tc.expectedValue, value)
			}

		})
	}
}

// func (suite *RedisConfigleamSecretsRepositorySuite) TestUpsertSecrets1() {
// 	type prePopulateData struct {
// 		key   string
// 		value interface{}
// 	}

// 	testCases := []struct {
// 		name        string
// 		prePopulate []prePopulateData
// 		inputValue  interface{}
// 		inputKey    string
// 		inputEnv    string
// 		expectedErr error
// 	}{
// 		{
// 			name:        "Upserting simple new key",
// 			inputEnv:    "develop",
// 			inputKey:    "key",
// 			inputValue:  "value",
// 			prePopulate: []prePopulateData{},
// 			expectedErr: nil,
// 		},
// 		{
// 			name:     "Upserting nested map with single key",
// 			inputKey: "key",
// 			inputEnv: "develop",
// 			inputValue: map[string]interface{}{
// 				"key1": map[string]interface{}{
// 					"key2": "value2",
// 				},
// 			},
// 			prePopulate: []prePopulateData{},
// 			expectedErr: nil,
// 		},
// 		{
// 			name:     "Upserting new key with deeply nested map",
// 			inputEnv: "develop",
// 			inputKey: "key",
// 			inputValue: map[string]interface{}{
// 				"key1": map[string]interface{}{
// 					"key2": map[string]interface{}{
// 						"key3": "value3",
// 					},
// 				},
// 			},
// 			prePopulate: []prePopulateData{},
// 			expectedErr: nil,
// 		},
// 		{
// 			name:     "Upserting nested map with existing key",
// 			inputEnv: "develop",
// 			inputKey: "key",
// 			inputValue: map[string]interface{}{
// 				"key1": map[string]interface{}{
// 					"key2": "updated_value",
// 				},
// 			},
// 			prePopulate: []prePopulateData{
// 				{key: "key", value: map[string]interface{}{
// 					"key1": map[string]interface{}{
// 						"key2": "value2",
// 					},
// 				}},
// 			},
// 			expectedErr: nil,
// 		},
// 		{
// 			name:     "Upserting deeply nested map with existing key",
// 			inputEnv: "develop",
// 			inputKey: "key",
// 			inputValue: map[string]interface{}{
// 				"key1": map[string]interface{}{
// 					"key2": map[string]interface{}{
// 						"key3": "updated_value",
// 					},
// 				},
// 			},
// 			prePopulate: []prePopulateData{
// 				{key: "key", value: map[string]interface{}{
// 					"key1": map[string]interface{}{
// 						"key2": map[string]interface{}{
// 							"key3": "value3",
// 						},
// 					},
// 				}},
// 			},
// 			expectedErr: nil,
// 		},
// 		{
// 			name:        "Upserting new key with array value",
// 			inputEnv:    "develop",
// 			inputKey:    "key",
// 			inputValue:  []interface{}{"value1", "value2", "value3"},
// 			prePopulate: []prePopulateData{},
// 			expectedErr: nil,
// 		},
// 		{
// 			name:       "Upserting new key with existing array value",
// 			inputEnv:   "develop",
// 			inputKey:   "key",
// 			inputValue: []interface{}{"updated_value1", "value2", "value3"},
// 			prePopulate: []prePopulateData{
// 				{key: "key", value: []interface{}{"value1", "value2", "value3"}},
// 			},
// 			expectedErr: nil,
// 		},
// 		{
// 			name:       "Upserting new key with existing non-map value",
// 			inputEnv:   "develop",
// 			inputKey:   "key",
// 			inputValue: "updated_value",
// 			prePopulate: []prePopulateData{
// 				{key: "key", value: "value"},
// 			},
// 			expectedErr: nil,
// 		},
// 		{
// 			name:        "Upserting new key with nil value",
// 			inputEnv:    "develop",
// 			inputKey:    "key",
// 			inputValue:  nil,
// 			prePopulate: []prePopulateData{},
// 			expectedErr: errors.New("'nil' can not be used as value for the key 'key'"),
// 		},
// 		{
// 			name:        "Upserting new key with invalid key path",
// 			inputEnv:    "develop",
// 			inputKey:    "invalid.key",
// 			inputValue:  "value",
// 			prePopulate: []prePopulateData{},
// 			expectedErr: fmt.Errorf("the key 'invalid.key' is malformed"),
// 		},
// 	}

// 	for _, tc := range testCases {
// 		suite.Run(tc.name, func() {
// 			// arrange
// 			suite.BeforeTest(tc.name)
// 			ctx := context.Background()

// 			// Pre-populate Redis with test data
// 			for _, data := range tc.prePopulate {
// 				value, err := json.Marshal(data.value)
// 				suite.Require().NoError(err)

// 				encrypted, err := suite.encryptor.Encrypt(ctx, value)
// 				suite.Require().NoError(err)

// 				err = suite.client.Set(ctx, suite.repository.GetSecretKey(tc.inputEnv, data.key), encrypted, 0).Err()
// 				suite.Require().NoError(err)
// 			}

// 			keys, err := suite.client.Keys(ctx, "*").Result()
// 			suite.Require().NoError(err)
// 			suite.Require().Equal(len(keys), len(tc.prePopulate))

// 			// act
// 			err = suite.repository.UpsertSecrets(ctx, tc.inputEnv, tc.inputKey, tc.inputValue)

// 			// assert
// 			if tc.expectedErr != nil {
// 				suite.Assert().Equal(tc.expectedErr, err)

// 				for _, pre := range tc.prePopulate {
// 					value, err := suite.repository.GetSecret(context.Background(), tc.inputEnv, pre.key)
// 					suite.Require().NoError(err)
// 					suite.Assert().Equal(value, pre.value)
// 				}
// 			} else {
// 				suite.Assert().NoError(err)

// 				value, err := suite.repository.GetSecret(context.Background(), tc.inputEnv, tc.inputKey)
// 				suite.Require().NoError(err)

// 				suite.Assert().Equal(tc.inputValue, value)
// 			}
// 		})
// 	}
// }

func (suite *RedisConfigleamSecretsRepositorySuite) TestUpsertSecrets() {
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
			// Arrange
			suite.BeforeTest(tc.name)
			ctx := context.Background()

			// Pre-populate Redis with test data
			for _, data := range tc.prePopulate {
				value, err := json.Marshal(data.value)
				suite.Require().NoError(err)

				encrypted, err := suite.encryptor.Encrypt(ctx, value)
				suite.Require().NoError(err)

				err = suite.client.Set(ctx, suite.repository.GetSecretKey(tc.inputEnv, data.key), encrypted, 0).Err()
				suite.Require().NoError(err)
			}

			keys, err := suite.client.Keys(ctx, "*").Result()
			suite.Require().NoError(err)
			suite.Require().Equal(len(keys), len(tc.prePopulate))

			// Act
			err = suite.repository.UpsertSecrets(ctx, tc.inputEnv, tc.inputValue)

			// Assert
			if tc.expectedErr != nil {
				suite.Assert().Equal(tc.expectedErr, err)

				for _, pre := range tc.prePopulate {
					value, err := suite.repository.GetSecret(context.Background(), tc.inputEnv, pre.key)
					suite.Require().NoError(err)
					suite.Assert().Equal(value, pre.value)
				}
			} else {
				suite.Assert().NoError(err)

				// Verify that the inserted secrets match the expected values
				for key, value := range tc.inputValue {
					expectedValue := value
					if expectedValue == nil {
						expectedValue = "nil" // Handle nil value
					}

					// Get the stored secret from Redis and decrypt it
					encrypted, err := suite.client.Get(ctx, suite.repository.GetSecretKey(tc.inputEnv, key)).Bytes()
					suite.Require().NoError(err)

					decrypted, err := suite.encryptor.Decrypt(ctx, encrypted)
					suite.Require().NoError(err)

					// Unmarshal the decrypted secret to compare with the expected value
					var storedValue interface{}
					err = json.Unmarshal([]byte(decrypted), &storedValue)
					suite.Require().NoError(err)

					suite.Assert().Equal(expectedValue, storedValue)
				}
			}
		})
	}
}

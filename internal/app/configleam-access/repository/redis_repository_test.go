package repository_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/raw-leak/configleam/internal/app/configleam-access/keys"
	"github.com/raw-leak/configleam/internal/app/configleam-access/repository"
	"github.com/raw-leak/configleam/internal/pkg/encryptor"
	"github.com/raw-leak/configleam/internal/pkg/permissions"
	rds "github.com/raw-leak/configleam/internal/pkg/redis"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
)

type RedisConfigleamAccessRepositorySuite struct {
	suite.Suite

	repository *repository.RedisRepository
	client     *redis.Client
	keys       *keys.Keys

	encryptor *encryptor.Encryptor
	key       string
}

func TestRedisRepositorySuite(t *testing.T) {
	suite.Run(t, new(RedisConfigleamAccessRepositorySuite))
}

func (suite *RedisConfigleamAccessRepositorySuite) SetupSuite() {
	var err error

	suite.client = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	suite.keys = keys.New()
	suite.key = "01234567890123456789012345678901"
	suite.encryptor, err = encryptor.NewEncryptor(suite.key)
	suite.Require().NoError(err)

	suite.repository = repository.NewRedisRepository(&rds.Redis{Client: suite.client}, suite.encryptor)
}

func (suite *RedisConfigleamAccessRepositorySuite) TearDownSuite() {
	suite.client.Close()
}

func (suite *RedisConfigleamAccessRepositorySuite) BeforeTest(testName string) {
	err := suite.client.FlushAll(context.Background()).Err()
	suite.Require().NoError(err)
	suite.Require().NoErrorf(err, "Flushing all data from redis before each test within the test: %s", testName)
}

func (suite *RedisConfigleamAccessRepositorySuite) TestStoreKeyWithPermissions() {
	ctx := context.Background()

	testCases := []struct {
		name        string
		key         string
		permissions permissions.AccessKeyPermissions
		meta        map[string]string
		expectErr   bool
	}{
		{
			name: "Store permissions with a single environment with meta and no admin successfully",
			key:  "test-access-key-1",
			permissions: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"default": permissions.ReadConfig | permissions.RevealSecrets,
				},
			},
			meta:      map[string]string{"name": "some name", "description:": "some description"},
			expectErr: false,
		},
		{
			name: "Store permissions with multiple environments with meta and admin successfully",
			key:  "test-access-key-2",
			meta: map[string]string{"name": "some name", "description:": "some description"},
			permissions: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"develop":    permissions.ReadConfig,
					"production": permissions.ReadConfig | permissions.EnvAdminAccess,
					"release":    permissions.ReadConfig | permissions.RevealSecrets,
				},
			},
			expectErr: false,
		},
		{
			name: "Store permissions with a single environment and no admin without meta successfully",
			key:  "test-access-key-1",
			meta: map[string]string{},
			permissions: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"default": permissions.ReadConfig | permissions.RevealSecrets,
				},
			},
			expectErr: false,
		},
		{
			name: "Store permissions with multiple environments and admin without meta successfully",
			key:  "test-access-key-2",
			meta: map[string]string{},
			permissions: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"develop":    permissions.ReadConfig,
					"production": permissions.ReadConfig | permissions.EnvAdminAccess,
					"release":    permissions.ReadConfig | permissions.RevealSecrets,
				},
			},
			expectErr: false,
		},
		{
			name: "Store permissions with admin permission successfully",
			meta: map[string]string{"name": "some name", "description:": "some description"},
			key:  "test-access-key-3",
			permissions: permissions.AccessKeyPermissions{
				Admin: true,
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.BeforeTest(tc.name)

			err := suite.repository.StoreKeyWithPermissions(ctx, tc.key, tc.permissions, tc.meta)
			if tc.expectErr {
				suite.Error(err)
			} else {
				suite.NoError(err)

				// ensure the key has been stored
				bytes, err := suite.client.Get(ctx, suite.repository.GetAccessKeyKey(tc.key)).Bytes()
				suite.NoError(err)

				decrypted, err := suite.encryptor.Decrypt(ctx, bytes)
				suite.NoError(err)

				var accessKeyPerms permissions.AccessKeyPermissions
				err = json.Unmarshal(decrypted, &accessKeyPerms)
				suite.NoError(err)

				suite.Equal(tc.permissions, accessKeyPerms)

				if tc.meta != nil {

					// ensure the meta has been stored
					meta, err := suite.client.HGetAll(ctx, suite.repository.GetAccessMetaKey(tc.key)).Result()
					suite.NoError(err)

					suite.Equal(tc.meta, meta)
				} else {
					bytes, err := suite.client.Get(ctx, suite.repository.GetAccessKeyKey(tc.key)).Bytes()
					suite.Error(err, redis.Nil)
					suite.Len(bytes, 0)
				}
			}
		})
	}
}

func (suite *RedisConfigleamAccessRepositorySuite) TestGetKeyPermissions() {
	type prePopulateData struct {
		key   string
		value permissions.AccessKeyPermissions
	}
	testCases := []struct {
		name           string
		key            string
		prePopulate    []prePopulateData
		expectedPerms  *permissions.AccessKeyPermissions
		expectErr      error
		expectedExists bool
	}{
		{
			name: "Retrieving successfully access key with permissions to a single environment access when it is the only key that exists",
			key:  "test-access-key",
			prePopulate: []prePopulateData{
				{
					key: "test-access-key", value: permissions.AccessKeyPermissions{
						Admin:       true,
						Permissions: permissions.Permissions{"develop": permissions.ReadConfig},
					},
				},
			},
			expectedPerms: &permissions.AccessKeyPermissions{
				Admin:       true,
				Permissions: permissions.Permissions{"develop": permissions.ReadConfig},
			},
			expectedExists: true,
		},
		{
			name: "Retrieving successfully access key with permissions to multiple environments access when it is the only key that exists",
			key:  "test-access-key",
			prePopulate: []prePopulateData{
				{
					key: "test-access-key", value: permissions.AccessKeyPermissions{
						Admin: true,
						Permissions: permissions.Permissions{
							"production": permissions.CreateSecrets,
							"develop":    permissions.RevealSecrets,
							"release":    permissions.CloneEnvironment | permissions.RevealSecrets,
						},
					},
				},
			},
			expectedPerms: &permissions.AccessKeyPermissions{
				Admin: true,
				Permissions: permissions.Permissions{
					"production": permissions.CreateSecrets,
					"develop":    permissions.RevealSecrets,
					"release":    permissions.CloneEnvironment | permissions.RevealSecrets,
				},
			},
			expectedExists: true,
		},
		{
			name: "Retrieving successfully access key with permissions to multiple environments access when there are other access keys",
			key:  "test-access-key",
			prePopulate: []prePopulateData{
				{
					key: "test-access-key", value: permissions.AccessKeyPermissions{
						Admin: true,
						Permissions: permissions.Permissions{
							"develop": permissions.RevealSecrets,
							"release": permissions.CloneEnvironment | permissions.RevealSecrets,
						},
					},
				},
				{
					key: "test-access-key-1", value: permissions.AccessKeyPermissions{
						Admin: false,
						Permissions: permissions.Permissions{
							"production": permissions.CreateSecrets,
							"release":    permissions.CloneEnvironment | permissions.RevealSecrets,
						},
					},
				},
			},
			expectedPerms: &permissions.AccessKeyPermissions{
				Admin: true,
				Permissions: permissions.Permissions{
					"develop": permissions.RevealSecrets,
					"release": permissions.CloneEnvironment | permissions.RevealSecrets,
				},
			},
			expectedExists: true,
		},
		{
			name: "Retrieving access key that does not exist to when there are other access keys",
			key:  "test-access-key-unexisting",
			prePopulate: []prePopulateData{
				{
					key: "test-access-key", value: permissions.AccessKeyPermissions{
						Admin: true,
						Permissions: permissions.Permissions{
							"develop": permissions.RevealSecrets,
							"release": permissions.CloneEnvironment | permissions.RevealSecrets,
						},
					},
				},
				{
					key: "test-access-key-1", value: permissions.AccessKeyPermissions{
						Admin: false,
						Permissions: permissions.Permissions{
							"production": permissions.CreateSecrets,
							"release":    permissions.CloneEnvironment | permissions.RevealSecrets,
						},
					},
				},
			},
			expectedPerms:  nil,
			expectedExists: false,
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

				err = suite.client.Set(ctx, suite.repository.GetAccessKeyKey(data.key), encrypted, 0).Err()
				suite.Require().NoError(err)
			}

			// Act
			perms, exists, err := suite.repository.GetKeyPermissions(ctx, tc.key)

			// Assert
			if tc.expectErr != nil {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)

				suite.Equal(tc.expectedExists, exists)
				suite.Equal(tc.expectedPerms, perms)

				data, err := json.Marshal(permissions.AccessKeyPermissions{
					Admin:       true,
					Permissions: permissions.Permissions{"default": permissions.ReadConfig},
				})
				suite.Require().NoError(err)

				encrypted, err := suite.encryptor.Encrypt(ctx, data)
				suite.NoError(err)

				suite.NoError(suite.client.Set(ctx, suite.repository.GetAccessKeyKey("test-access-key"), encrypted, 0).Err())

			}
		})
	}
}

func (suite *RedisConfigleamAccessRepositorySuite) TestRemoveKeys() {
	type prePopulateData struct {
		key   string
		value permissions.AccessKeyPermissions
	}
	testCases := []struct {
		name        string
		keys        []string
		leftKeys    []string
		prePopulate []prePopulateData
		expectErr   error
	}{
		{
			name: "Removing successfully a single existing key",
			keys: []string{"test-access-key"},
			prePopulate: []prePopulateData{
				{
					key: "test-access-key", value: permissions.AccessKeyPermissions{
						Admin:       true,
						Permissions: permissions.Permissions{"develop": permissions.ReadConfig},
					},
				},
			},
			expectErr: nil,
		},
		{
			name: "Removing successfully multiple existing key",
			keys: []string{"test-access-key-1", "test-access-key-2", "test-access-key-3", "test-access-key-4"},
			prePopulate: []prePopulateData{
				{
					key: "test-access-key-1", value: permissions.AccessKeyPermissions{
						Admin: false,
						Permissions: permissions.Permissions{
							"develop": permissions.ReadConfig,
							"release": permissions.CloneEnvironment,
						},
					},
				},
				{
					key: "test-access-key-2", value: permissions.AccessKeyPermissions{
						Admin: false,
						Permissions: permissions.Permissions{
							"develop": permissions.ReadConfig,
						},
					},
				},
				{
					key: "test-access-key-3", value: permissions.AccessKeyPermissions{
						Admin: true,
						Permissions: permissions.Permissions{
							"release": permissions.CloneEnvironment,
						},
					},
				},
				{
					key: "test-access-key-4", value: permissions.AccessKeyPermissions{
						Admin: false,
						Permissions: permissions.Permissions{
							"develop": permissions.ReadConfig,
							"release": permissions.CloneEnvironment,
						},
					},
				},
			},
			expectErr: nil,
		},
		{
			name: "Removing successfully two keys when four exist",
			keys: []string{"test-access-key-1", "test-access-key-2"},
			prePopulate: []prePopulateData{
				{
					key: "test-access-key-1", value: permissions.AccessKeyPermissions{
						Admin: false,
						Permissions: permissions.Permissions{
							"develop": permissions.ReadConfig,
							"release": permissions.CloneEnvironment,
						},
					},
				},
				{
					key: "test-access-key-2", value: permissions.AccessKeyPermissions{
						Admin: false,
						Permissions: permissions.Permissions{
							"develop": permissions.ReadConfig,
						},
					},
				},
				{
					key: "test-access-key-3", value: permissions.AccessKeyPermissions{
						Admin: true,
						Permissions: permissions.Permissions{
							"release": permissions.CloneEnvironment,
						},
					},
				},
				{
					key: "test-access-key-4", value: permissions.AccessKeyPermissions{
						Admin: false,
						Permissions: permissions.Permissions{
							"develop": permissions.ReadConfig,
							"release": permissions.CloneEnvironment,
						},
					},
				},
			},
			leftKeys:  []string{"test-access-key-3", "test-access-key-4"},
			expectErr: nil,
		},
		{
			name: "Removing two keys that does not exist and there are four keys left",
			keys: []string{"test-access-key-unexisting-1", "test-access-key-unexisting-2"},
			prePopulate: []prePopulateData{
				{
					key: "test-access-key-1", value: permissions.AccessKeyPermissions{
						Admin: false,
						Permissions: permissions.Permissions{
							"develop": permissions.ReadConfig,
							"release": permissions.CloneEnvironment,
						},
					},
				},
				{
					key: "test-access-key-2", value: permissions.AccessKeyPermissions{
						Admin: false,
						Permissions: permissions.Permissions{
							"develop": permissions.ReadConfig,
						},
					},
				},
				{
					key: "test-access-key-3", value: permissions.AccessKeyPermissions{
						Admin: true,
						Permissions: permissions.Permissions{
							"release": permissions.CloneEnvironment,
						},
					},
				},
				{
					key: "test-access-key-4", value: permissions.AccessKeyPermissions{
						Admin: false,
						Permissions: permissions.Permissions{
							"develop": permissions.ReadConfig,
							"release": permissions.CloneEnvironment,
						},
					},
				},
			},
			leftKeys:  []string{"test-access-key-1", "test-access-key-2", "test-access-key-3", "test-access-key-4"},
			expectErr: nil,
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

				err = suite.client.Set(ctx, suite.repository.GetAccessKeyKey(data.key), encrypted, 0).Err()
				suite.Require().NoError(err)
			}

			// Act
			err := suite.repository.RemoveKeys(ctx, tc.keys)

			// Assert
			if tc.expectErr != nil {
				suite.Error(err)
			} else {
				suite.NoError(err)

				for _, key := range tc.keys {
					key, err := suite.client.Get(ctx, suite.repository.GetAccessKeyKey(key)).Result()
					suite.Error(err)
					suite.Equal(err, redis.Nil)

					suite.Assert().Equal(key, "")
				}

				for _, key := range tc.leftKeys {
					key, err := suite.client.Get(ctx, suite.repository.GetAccessKeyKey(key)).Result()
					suite.Assert().NoError(err)
					suite.Assert().NotZero(key)
				}
			}
		})
	}
}

package repository_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/raw-leak/configleam/internal/app/access/keys"
	"github.com/raw-leak/configleam/internal/app/access/repository"
	"github.com/raw-leak/configleam/internal/pkg/encryptor"
	"github.com/raw-leak/configleam/internal/pkg/permissions"
	rds "github.com/raw-leak/configleam/internal/pkg/redis"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
)

type RedisAccessRepositorySuite struct {
	suite.Suite

	repository *repository.RedisRepository
	client     *redis.Client
	keys       *keys.Keys

	encryptor *encryptor.Encryptor
	key       string
}

func TestRedisRepositorySuite(t *testing.T) {
	suite.Run(t, new(RedisAccessRepositorySuite))
}

func (suite *RedisAccessRepositorySuite) SetupSuite() {
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

func (suite *RedisAccessRepositorySuite) TearDownSuite() {
	suite.client.Close()
}

func (suite *RedisAccessRepositorySuite) BeforeTest(testName string) {
	err := suite.client.FlushAll(context.Background()).Err()
	suite.Require().NoError(err)
	suite.Require().NoErrorf(err, "Flushing all data from redis before each test within the test: %s", testName)
}

func (suite *RedisAccessRepositorySuite) TestStoreAccessKey() {
	ctx := context.Background()

	testCases := []struct {
		name      string
		inputDate repository.AccessKey
		expectErr bool
	}{
		{
			name: "Store permissions with a single environment with meta and no permissions.admin successfully",
			inputDate: repository.AccessKey{
				Key: "test-access-key-1",
				Perms: permissions.AccessKeyPermissions{
					Admin: false,
					Permissions: permissions.Permissions{
						"default": permissions.ReadConfig | permissions.RevealSecrets,
					},
				},
				Metadata: repository.AccessKeyMetadata{
					CreationDate: time.Now(),
				},
			},
			expectErr: false,
		},
		{
			name: "Store permissions with multiple environments with meta and permissions.admin successfully",
			inputDate: repository.AccessKey{
				Key: "test-access-key-2",
				Perms: permissions.AccessKeyPermissions{
					Admin: false,
					Permissions: permissions.Permissions{
						"develop":    permissions.ReadConfig,
						"production": permissions.ReadConfig | permissions.EnvAdminAccess,
						"release":    permissions.ReadConfig | permissions.RevealSecrets,
					},
				},
				Metadata: repository.AccessKeyMetadata{
					CreationDate: time.Now(),
				},
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.BeforeTest(tc.name)

			err := suite.repository.StoreAccessKey(ctx, tc.inputDate)

			if tc.expectErr {
				suite.Error(err)
			} else {
				suite.NoError(err)

				// determine encrypted key
				encryptedKeyBytes, err := suite.encryptor.EncryptDet(ctx, []byte(tc.inputDate.Key))
				suite.NoError(err)

				encryptedKey := base64.StdEncoding.EncodeToString(encryptedKeyBytes)

				// ensure the key has been stored
				rawPerms, err := suite.client.Get(ctx, suite.repository.GetAccessKeyKey(encryptedKey)).Bytes()
				suite.NoError(err)

				decryptedPermsBytes, err := suite.encryptor.Decrypt(ctx, rawPerms)
				suite.NoError(err)

				var accessKeyPerms permissions.AccessKeyPermissions
				err = json.Unmarshal(decryptedPermsBytes, &accessKeyPerms)
				suite.NoError(err)
				suite.Equal(tc.inputDate.Perms, accessKeyPerms)

				// ensure the meta has been stored
				metaBytes, err := suite.client.Get(ctx, suite.repository.GetAccessMetaKey(encryptedKey)).Bytes()
				suite.NoError(err)

				var meta repository.AccessKeyMetadata
				err = json.Unmarshal(metaBytes, &meta)
				suite.NoError(err)
				suite.Equal(tc.inputDate.Metadata.CreationDate.Unix(), meta.CreationDate.Unix())

				// ensure that the key has been added to the sorted-set
				score, err := suite.client.ZScore(ctx, suite.repository.GetAccessSetKey(), encryptedKey).Result()
				suite.NotEqual(err, redis.Nil)
				suite.NoError(err)

				// check if the score is equal to the creation date of the access key
				suite.Equal(float64(tc.inputDate.Metadata.CreationDate.Unix()), score)
			}
		})
	}
}

func (suite *RedisAccessRepositorySuite) TestGetAccessKeyPermissions() {
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

				encryptedPermsBytes, err := suite.encryptor.Encrypt(ctx, value)
				suite.Require().NoError(err)

				encryptedKeyBytes, err := suite.encryptor.EncryptDet(ctx, []byte(data.key))
				suite.Require().NoError(err)

				encryptedKey := base64.StdEncoding.EncodeToString(encryptedKeyBytes)

				err = suite.client.Set(ctx, suite.repository.GetAccessKeyKey(encryptedKey), encryptedPermsBytes, 0).Err()
				suite.Require().NoError(err)
			}

			// Act
			perms, exists, err := suite.repository.GetAccessKeyPermissions(ctx, tc.key)

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

func (suite *RedisAccessRepositorySuite) TestRemoveKeys() {
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

func (suite *RedisAccessRepositorySuite) TestPaginateAccessKeys() {
	ctx := context.Background()

	sampleAccessKeys := []repository.AccessKey{
		{
			Key: "cfg_test-access-key1",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"develop": permissions.ReadConfig | permissions.RevealSecrets,
					"prod":    permissions.AccessDashboard | permissions.EnvAdminAccess,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key1",
				MaskedKey:      "cfg_test****key1",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, 0, -1),
			},
		},
		{
			Key: "cfg_test-access-key2",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"release": permissions.ReadConfig | permissions.RevealSecrets,
					"develop": permissions.ReadConfig | permissions.CloneEnvironment,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key2",
				MaskedKey:      "cfg_test****key1",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, 0, -2),
			},
		},
		{
			Key: "cfg_test-access-key3",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"develop": permissions.ReadConfig | permissions.RevealSecrets,
					"prod":    permissions.AccessDashboard | permissions.EnvAdminAccess,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key3",
				MaskedKey:      "cfg_test****key3",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, 0, -3),
			},
		},
		{
			Key: "cfg_test-access-key4",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"release": permissions.ReadConfig | permissions.RevealSecrets,
					"develop": permissions.ReadConfig | permissions.CloneEnvironment,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key4",
				MaskedKey:      "cfg_test****key4",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, 0, -10),
			},
		},
		{
			Key: "cfg_test-access-key5",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"qa":      permissions.ReadConfig | permissions.RevealSecrets,
					"staging": permissions.ReadConfig | permissions.CloneEnvironment,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key5",
				MaskedKey:      "cfg_test****key5",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, 0, -15),
			},
		},
		{
			Key: "cfg_test-access-key6",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"qa":      permissions.ReadConfig | permissions.RevealSecrets,
					"staging": permissions.ReadConfig | permissions.CloneEnvironment,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key6",
				MaskedKey:      "cfg_test****key6",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, 0, -25),
			},
		},
		{
			Key: "cfg_test-access-key7",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"qa":      permissions.ReadConfig | permissions.RevealSecrets,
					"staging": permissions.ReadConfig | permissions.CloneEnvironment,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key7",
				MaskedKey:      "cfg_test****key7",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, -1, 0),
			},
		},
		{
			Key: "cfg_test-access-key8",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"qa":      permissions.ReadConfig | permissions.RevealSecrets,
					"staging": permissions.ReadConfig | permissions.CloneEnvironment,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key8",
				MaskedKey:      "cfg_test****key8",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, -1, -1),
			},
		},
		{
			Key: "cfg_test-access-key9",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"qa":      permissions.ReadConfig | permissions.RevealSecrets,
					"staging": permissions.ReadConfig | permissions.CloneEnvironment,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key9",
				MaskedKey:      "cfg_test****key9",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, -1, -21),
			},
		},
		{
			Key: "cfg_test-access-key10",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"qa":      permissions.ReadConfig | permissions.RevealSecrets,
					"staging": permissions.ReadConfig | permissions.CloneEnvironment,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key10",
				MaskedKey:      "cfg_test****key10",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, -1, -25),
			},
		},
		{
			Key: "cfg_test-access-key11",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"qa":      permissions.ReadConfig | permissions.RevealSecrets,
					"staging": permissions.ReadConfig | permissions.CloneEnvironment,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key11",
				MaskedKey:      "cfg_test****key11",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, -1, -26),
			},
		},
		{
			Key: "cfg_test-access-key12",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"qa":      permissions.ReadConfig | permissions.RevealSecrets,
					"staging": permissions.ReadConfig | permissions.CloneEnvironment | permissions.AccessDashboard,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key12",
				MaskedKey:      "cfg_test****key12",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, -2, -2),
			},
		},
		{
			Key: "cfg_test-access-key13",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"qa":      permissions.ReadConfig | permissions.RevealSecrets,
					"staging": permissions.ReadConfig | permissions.CloneEnvironment | permissions.CreateSecrets,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key13",
				MaskedKey:      "cfg_test****key13",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, -2, -6),
			},
		},
		{
			Key: "cfg_test-access-key14",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"qa":      permissions.ReadConfig | permissions.RevealSecrets,
					"staging": permissions.ReadConfig | permissions.CloneEnvironment | permissions.CreateSecrets | permissions.AccessDashboard,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key14",
				MaskedKey:      "cfg_test****key14",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, -2, -15),
			},
		},
		{
			Key: "cfg_test-access-key15",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"qa":      permissions.ReadConfig | permissions.RevealSecrets,
					"staging": permissions.ReadConfig | permissions.CloneEnvironment | permissions.CreateSecrets | permissions.AccessDashboard | permissions.EnvAdminAccess,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key15",
				MaskedKey:      "cfg_test****key15",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, -3, -3),
			},
		},
		{
			Key: "cfg_test-access-key16",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"qa":      permissions.ReadConfig | permissions.RevealSecrets,
					"staging": permissions.ReadConfig | permissions.CloneEnvironment | permissions.CreateSecrets | permissions.AccessDashboard | permissions.EnvAdminAccess | permissions.Admin,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key16",
				MaskedKey:      "cfg_test****key16",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, -3, -23),
			},
		},
		{
			Key: "cfg_test-access-key17",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"qa":      permissions.ReadConfig | permissions.RevealSecrets,
					"staging": permissions.ReadConfig | permissions.CloneEnvironment | permissions.CreateSecrets | permissions.AccessDashboard | permissions.EnvAdminAccess | permissions.Admin | permissions.RevealSecrets,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key17",
				MaskedKey:      "cfg_test****key17",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, -4, -2),
			},
		},
		{
			Key: "cfg_test-access-key18",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"qa":      permissions.ReadConfig | permissions.RevealSecrets,
					"staging": permissions.ReadConfig | permissions.CloneEnvironment | permissions.CreateSecrets | permissions.AccessDashboard | permissions.EnvAdminAccess | permissions.Admin | permissions.RevealSecrets | permissions.CloneEnvironment,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key18",
				MaskedKey:      "cfg_test****key18",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, -4, -12),
			},
		},
		{
			Key: "cfg_test-access-key19",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"qa":      permissions.ReadConfig | permissions.RevealSecrets,
					"staging": permissions.ReadConfig | permissions.CloneEnvironment | permissions.CreateSecrets | permissions.AccessDashboard | permissions.EnvAdminAccess | permissions.Admin | permissions.RevealSecrets | permissions.CloneEnvironment | permissions.CreateSecrets,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key19",
				MaskedKey:      "cfg_test****key19",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, -4, -21),
			},
		},
		{
			Key: "cfg_test-access-key20",
			Perms: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: permissions.Permissions{
					"qa":      permissions.ReadConfig | permissions.RevealSecrets,
					"staging": permissions.ReadConfig | permissions.CloneEnvironment | permissions.CreateSecrets | permissions.AccessDashboard | permissions.EnvAdminAccess | permissions.Admin | permissions.RevealSecrets | permissions.CloneEnvironment | permissions.CreateSecrets | permissions.AccessDashboard,
				},
			},
			Metadata: repository.AccessKeyMetadata{
				Name:           "some name for key20",
				MaskedKey:      "cfg_test****key20",
				ExpirationDate: time.Time{},
				CreationDate:   time.Now().AddDate(0, -5, -1),
			},
		},
	}

	testCases := []struct {
		name        string
		page        int
		size        int
		expectedLen int // Expected length of the returned slice
		expectErr   bool
	}{
		{
			name:        "Retrieve first page with one element",
			page:        1,
			size:        1,
			expectedLen: 1,
			expectErr:   false,
		},
		{
			name:        "Retrieve second page with one element",
			page:        2,
			size:        1,
			expectedLen: 1,
			expectErr:   false,
		},
		{
			name:        "Retrieve 10th page with one element",
			page:        10,
			size:        1,
			expectedLen: 1,
			expectErr:   false,
		},
		{
			name:        "Retrieve first page with three element",
			page:        1,
			size:        10,
			expectedLen: 10,
			expectErr:   false,
		},
		{
			name:        "Retrieve ss page with three element",
			page:        3,
			size:        20,
			expectedLen: 0,
			expectErr:   false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.BeforeTest(tc.name)
			ctx = context.Background()

			for _, accessKey := range sampleAccessKeys {
				err := suite.repository.StoreAccessKey(ctx, accessKey)
				suite.NoError(err)
			}

			paginated, err := suite.repository.PaginateAccessKeys(ctx, tc.page, tc.size)

			if tc.expectErr {
				suite.Error(err)
			} else {
				suite.NoError(err)

				suite.Len(paginated.Items, tc.expectedLen)
				suite.Equal(paginated.Total, len(sampleAccessKeys))
				suite.Equal(paginated.Size, tc.size)
				suite.Equal(paginated.Page, tc.page)
				suite.Equal(paginated.Page, tc.page)
				suite.Equal(paginated.Pages, len(sampleAccessKeys)/tc.size)

				from := (tc.page - 1) * tc.size
				to := from + tc.size

				if from > len(sampleAccessKeys)-1 {
					from = len(sampleAccessKeys) - 1
				}

				if to > len(sampleAccessKeys)-1 {
					to = len(sampleAccessKeys) - 1
				}

				expectedKeys := sampleAccessKeys[from:to]

				suite.Len(paginated.Items, len(expectedKeys))

				for i, fetchedAccessKey := range paginated.Items {

					metadata := expectedKeys[i]

					suite.Equal(metadata.Metadata.CreationDate, fetchedAccessKey.CreationDate)
					suite.Equal(metadata.Metadata.ExpirationDate, fetchedAccessKey.ExpirationDate)
					suite.Equal(metadata.Metadata.MaskedKey, fetchedAccessKey.MaskedKey)
					suite.Equal(metadata.Metadata.Name, fetchedAccessKey.Name)
				}

			}
		})
	}
}

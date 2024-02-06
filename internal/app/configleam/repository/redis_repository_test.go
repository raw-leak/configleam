package repository_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/raw-leak/configleam/internal/app/configleam/repository"
	"github.com/raw-leak/configleam/internal/app/configleam/types"
	rds "github.com/raw-leak/configleam/internal/pkg/redis"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RedisConfigRepositorySuite struct {
	suite.Suite
	repository *repository.RedisConfigRepository
	client     *redis.Client
}

func (suite *RedisConfigRepositorySuite) SetupSuite() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	suite.client = client
	suite.repository = repository.NewRedisRepository(&rds.Redis{Client: client})
}

func (suite *RedisConfigRepositorySuite) TearDownSuite() {
	suite.client.Close()
}

func (suite *RedisConfigRepositorySuite) BeforeTest(suiteName, testName string) {
	suite.client.FlushAll(context.Background())
}

// TestUpsertConfig tests UpsertConfig method
func (suite *RedisConfigRepositorySuite) TestUpsertConfig() {
	type testCase struct {
		name   string
		env    string
		repo   string
		config *types.ParsedRepoConfig

		expectedErr     bool
		expectedGlobals []map[string]interface{}
		expectedGroups  []map[string]interface{}
		expectedKeys    []string
	}

	testCases := []testCase{
		{
			name: "Successful upsert",
			env:  "develop",
			repo: "test-repo",
			config: &types.ParsedRepoConfig{
				AllKeys: []string{},
				Groups: map[string]types.GroupConfig{
					"group:app1": {
						Local: map[string]interface{}{
							"local": map[string]interface{}{
								"port": 222,
							},
						},
						Global: []string{"globalKey1"},
					},
				},
				Globals: map[string]interface{}{"globalKey": "globalValue"},
			},
			expectedErr: false,
			expectedGroups: []map[string]interface{}{
				{
					"test-repo:develop:group:app1": map[string]interface{}{
						"Local": map[string]interface{}{
							"local": map[string]interface{}{
								"port": float64(222),
							},
						},
						"Global": []interface{}{"globalKey1"},
					},
				},
			},
			expectedGlobals: []map[string]interface{}{{"test-repo:develop:global:globalKey": "globalValue"}},
			expectedKeys:    []string{"test-repo:develop:group:app1", "test-repo:develop:global:globalKey"},
		},
		{
			name: "Upsert with empty configuration",
			env:  "test-env",
			repo: "empty-repo",
			config: &types.ParsedRepoConfig{
				AllKeys: []string{},
				Groups:  map[string]types.GroupConfig{},
				Globals: map[string]interface{}{},
			},
			expectedErr:     false,
			expectedGlobals: nil,
			expectedGroups:  nil,
			expectedKeys:    []string{},
		},
		{
			name: "Attempt to upsert non-existent env and repo",
			env:  "nonexistent-env",
			repo: "nonexistent-repo",
			config: &types.ParsedRepoConfig{
				AllKeys: []string{"nonexistent-key"},
				Groups: map[string]types.GroupConfig{
					"group:nonexistent": {
						Local:  map[string]interface{}{"key": "value"},
						Global: []string{"nonexistent-global"},
					},
				},
				Globals: map[string]interface{}{"nonexistent-global": "value"},
			},
			expectedErr: false,
			expectedGlobals: []map[string]interface{}{
				{"nonexistent-repo:nonexistent-env:global:nonexistent-global": "value"},
			},
			expectedGroups: []map[string]interface{}{
				{"nonexistent-repo:nonexistent-env:group:nonexistent": map[string]interface{}{
					"Local":  map[string]interface{}{"key": "value"},
					"Global": []interface{}{"nonexistent-global"},
				}},
			},
			expectedKeys: []string{
				"nonexistent-repo:nonexistent-env:group:nonexistent",
				"nonexistent-repo:nonexistent-env:global:nonexistent-global",
			},
		},
		{
			name: "Multiple globals and groups with diverse locals",
			env:  "production",
			repo: "complex-config-repo",
			config: &types.ParsedRepoConfig{
				AllKeys: []string{"globalKey1", "globalKey2", "globalKey3"},
				Groups: map[string]types.GroupConfig{
					"group:service1": {
						Local: map[string]interface{}{
							"servicePort": 8080,
							"servicePath": "/api/v1",
						},
						Global: []string{"globalKey1", "globalKey2"},
					},
					"group:service2": {
						Local: map[string]interface{}{
							"dbHost": "db.internal",
							"dbPort": 5432,
						},
						Global: []string{"globalKey2", "globalKey3"},
					},
				},
				Globals: map[string]interface{}{
					"globalKey1": map[string]interface{}{"apiUrl": "https://api.example.com"},
					"globalKey2": "shared-value",
					"globalKey3": []interface{}{"item1", "item2"},
				},
			},
			expectedErr: false,
			expectedGlobals: []map[string]interface{}{
				{"complex-config-repo:production:global:globalKey1": map[string]interface{}{"apiUrl": "https://api.example.com"}},
				{"complex-config-repo:production:global:globalKey2": "shared-value"},
				{"complex-config-repo:production:global:globalKey3": []interface{}{"item1", "item2"}},
			},
			expectedGroups: []map[string]interface{}{
				{"complex-config-repo:production:group:service1": map[string]interface{}{
					"Local": map[string]interface{}{
						"servicePort": float64(8080),
						"servicePath": "/api/v1",
					},
					"Global": []interface{}{"globalKey1", "globalKey2"},
				}},
				{"complex-config-repo:production:group:service2": map[string]interface{}{
					"Local": map[string]interface{}{
						"dbHost": "db.internal",
						"dbPort": float64(5432),
					},
					"Global": []interface{}{"globalKey2", "globalKey3"},
				}},
			},
			expectedKeys: []string{
				"complex-config-repo:production:group:service1",
				"complex-config-repo:production:group:service2",
				"complex-config-repo:production:global:globalKey1",
				"complex-config-repo:production:global:globalKey2",
				"complex-config-repo:production:global:globalKey3",
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := suite.repository.UpsertConfig(ctx, tc.env, tc.repo, tc.config)

			if tc.expectedErr {
				suite.Assert().Error(err)
				// ensure that NO keys has been generated
				keys, err := suite.client.Keys(ctx, tc.repo+":"+tc.env+":*").Result()
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), 0, len(keys))

			} else {
				// assert no error
				assert.NoError(suite.T(), err)

				// check globals
				for _, g := range tc.expectedGlobals {
					for key, expectedValue := range g {
						val, err := suite.client.Get(ctx, key).Result()
						assert.NoError(suite.T(), err)

						var actualValue interface{}
						err = json.Unmarshal([]byte(val), &actualValue)
						assert.NoError(suite.T(), err)

						assert.Equal(suite.T(), expectedValue, actualValue)
					}
				}

				// check groups
				for _, g := range tc.expectedGlobals {
					for key, expectedValue := range g {
						val, err := suite.client.Get(ctx, key).Result()
						assert.NoError(suite.T(), err)

						var actualValue interface{}
						err = json.Unmarshal([]byte(val), &actualValue)
						assert.NoError(suite.T(), err)

						assert.Equal(suite.T(), expectedValue, actualValue)
					}
				}

				// check globals
				for _, g := range tc.expectedGroups {
					for key, expectedValue := range g {
						val, err := suite.client.Get(ctx, key).Result()
						assert.NoError(suite.T(), err)

						var actualValue interface{}
						err = json.Unmarshal([]byte(val), &actualValue)
						assert.NoError(suite.T(), err)

						assert.Equal(suite.T(), expectedValue, actualValue)
					}
				}

				// ensure only expected keys exist
				keys, err := suite.client.Keys(ctx, tc.repo+":"+tc.env+":*").Result()
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), len(tc.expectedKeys), len(keys))
				for _, key := range keys {
					assert.Contains(suite.T(), tc.expectedKeys, key)
				}
			}
		})
	}
}

func TestRedisConfigRepositorySuite(t *testing.T) {
	suite.Run(t, new(RedisConfigRepositorySuite))
}

func (suite *RedisConfigRepositorySuite) TestReadConfig() {
	type prePopulateData struct {
		key   string
		value interface{}
	}

	type testCase struct {
		name           string
		env            string
		groups         []string
		globalKeys     []string
		prePopulate    []prePopulateData // Data to pre-populate in Redis
		expectedResult map[string]interface{}
		expectedErr    bool
	}

	testCases := []testCase{
		{
			name:       "ReadConfig with populated groups and globals",
			env:        "develop",
			groups:     []string{"app1", "app2"},
			globalKeys: []string{"globalKey1", "globalKey2"},
			prePopulate: []prePopulateData{
				{"test-repo:develop:group:app1", types.GroupConfig{Local: map[string]interface{}{"localKey1": "localValue1"}, Global: []string{"globalKey1"}}},
				{"test-repo:develop:group:app2", types.GroupConfig{Local: map[string]interface{}{"localKey2": "localValue2"}, Global: []string{"globalKey2"}}},
				{"test-repo:develop:global:globalKey1", "globalValue1"},
				{"test-repo:develop:global:globalKey2", "globalValue2"},
			},
			expectedResult: map[string]interface{}{
				"app1": map[string]interface{}{
					"localKey1":  "localValue1",
					"globalKey1": "globalValue1",
				},
				"app2": map[string]interface{}{
					"localKey2":  "localValue2",
					"globalKey2": "globalValue2",
				},
				"globalKey1": "globalValue1",
				"globalKey2": "globalValue2",
			},
			expectedErr: false,
		},
		{
			name:       "Complex nested variables and multiple groups",
			env:        "staging",
			groups:     []string{"service1", "service2"},
			globalKeys: []string{"globalConfig1", "nestedGlobalConfig"},
			prePopulate: []prePopulateData{
				{"staging-repo:staging:group:service1", types.GroupConfig{
					Local: map[string]interface{}{
						"nestedLocal": map[string]interface{}{
							"subKey1": "subValue1",
							"subKey2": map[string]interface{}{
								"subSubKey1": "subSubValue1",
							},
						},
					},
					Global: []string{"globalConfig1"},
				}},
				{"staging-repo:staging:group:service2", types.GroupConfig{
					Local: map[string]interface{}{
						"serviceConfig": map[string]interface{}{
							"port":    float64(8080),
							"timeout": float64(30),
						},
					},
					Global: []string{"nestedGlobalConfig"},
				}},
				{"staging-repo:staging:global:globalConfig1", "globalValue1"},
				{"staging-repo:staging:global:nestedGlobalConfig", map[string]interface{}{
					"globalNestedKey": map[string]interface{}{
						"nestedKey1": "nestedValue1",
						"nestedKey2": float64(42),
					},
				}},
			},
			expectedResult: map[string]interface{}{
				"service1": map[string]interface{}{
					"nestedLocal": map[string]interface{}{
						"subKey1": "subValue1",
						"subKey2": map[string]interface{}{
							"subSubKey1": "subSubValue1",
						},
					},
					"globalConfig1": "globalValue1",
				},
				"service2": map[string]interface{}{
					"serviceConfig": map[string]interface{}{
						"port":    float64(8080),
						"timeout":  float64(30),
					},
					"nestedGlobalConfig": map[string]interface{}{
						"globalNestedKey": map[string]interface{}{
							"nestedKey1": "nestedValue1",
							"nestedKey2":  float64(42),
						},
					},
				},
			},
			expectedErr: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ctx := context.Background()

			// Pre-populate Redis with test data
			for _, data := range tc.prePopulate {
				value, err := json.Marshal(data.value)
				suite.Require().NoError(err)
				err = suite.client.Set(ctx, data.key, value, 0).Err()
				suite.Require().NoError(err)
			}

			// Call ReadConfig and assert the results
			result, err := suite.repository.ReadConfig(ctx, tc.env, tc.groups, tc.globalKeys)

			if tc.expectedErr {
				suite.Assert().Error(err)
			} else {
				suite.Assert().NoError(err)
				suite.Assert().Equal(tc.expectedResult, result)
			}
		})
	}
}

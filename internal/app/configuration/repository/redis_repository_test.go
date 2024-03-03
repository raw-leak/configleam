package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/raw-leak/configleam/internal/app/configuration/repository"
	"github.com/raw-leak/configleam/internal/app/configuration/types"
	rds "github.com/raw-leak/configleam/internal/pkg/redis"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RedisRepositorySuite struct {
	suite.Suite
	repository *repository.RedisRepository
	client     *redis.Client
}

func TestRedisRepositorySuite(t *testing.T) {
	suite.Run(t, new(RedisRepositorySuite))
}

func (suite *RedisRepositorySuite) SetupSuite() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	suite.client = client
	suite.repository = repository.NewRedisRepository(&rds.Redis{Client: client})
}

func (suite *RedisRepositorySuite) TearDownSuite() {
	suite.client.Close()
}

func (suite *RedisRepositorySuite) BeforeTest(testName string) {
	err := suite.client.FlushAll(context.Background()).Err()
	assert.NoErrorf(suite.T(), err, "Flushing all data from redis before each test within the test: %s", testName)
}

func (suite *RedisRepositorySuite) TestUpsertConfig() {
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
							"servicePath": "/api",
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
						"servicePath": "/api",
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
			suite.BeforeTest(tc.name)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := suite.repository.UpsertConfig(ctx, tc.env, tc.repo, tc.config)

			if tc.expectedErr {
				suite.Assert().Error(err)
				// ensure that NO keys has been generated
				keys, err := suite.client.Keys(ctx, repository.ConfigurationPrefix+":"+tc.repo+":"+tc.env+":*").Result()
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), 0, len(keys))

			} else {
				// assert no error
				assert.NoError(suite.T(), err)

				// check globals
				for _, g := range tc.expectedGlobals {
					for key, expectedValue := range g {
						val, err := suite.client.Get(ctx, repository.ConfigurationPrefix+":"+key).Result()
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
						val, err := suite.client.Get(ctx, repository.ConfigurationPrefix+":"+key).Result()
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
						val, err := suite.client.Get(ctx, repository.ConfigurationPrefix+":"+key).Result()
						assert.NoError(suite.T(), err)

						var actualValue interface{}
						err = json.Unmarshal([]byte(val), &actualValue)
						assert.NoError(suite.T(), err)

						assert.Equal(suite.T(), expectedValue, actualValue)
					}
				}

				// ensure only expected keys exist
				keys, err := suite.client.Keys(ctx, repository.ConfigurationPrefix+":"+tc.repo+":"+tc.env+":*").Result()
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), len(tc.expectedKeys), len(keys))

				expectedFullKeys := []string{}
				for _, k := range tc.expectedKeys {
					fullExpectedKey := repository.ConfigurationPrefix + ":" + k
					expectedFullKeys = append(expectedFullKeys, fullExpectedKey)
				}

				for _, key := range keys {
					suite.Contains(expectedFullKeys, key)
				}
			}
		})
	}
}

func (suite *RedisRepositorySuite) TestReadConfig() {
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
						"timeout": float64(30),
					},
					"nestedGlobalConfig": map[string]interface{}{
						"globalNestedKey": map[string]interface{}{
							"nestedKey1": "nestedValue1",
							"nestedKey2": float64(42),
						},
					},
				},
				"globalConfig1": "globalValue1",
				"nestedGlobalConfig": map[string]interface{}{
					"globalNestedKey": map[string]interface{}{
						"nestedKey1": "nestedValue1",
						"nestedKey2": float64(42),
					},
				},
			},
			expectedErr: false,
		},
		{
			name:       "Complex Scenario with Overlapping Keys and Nested Groups",
			env:        "production",
			groups:     []string{"service1", "service2", "service3"},
			globalKeys: []string{"globalKey4", "globalKey5", "globalKey6"},
			prePopulate: []prePopulateData{
				// production
				{"repo1:production:group:service1", types.GroupConfig{
					Local: map[string]interface{}{
						"nestedConfig": map[string]interface{}{
							"subKey1": "subValue1",
							"subKey2": map[string]interface{}{
								"subSubKey1": "subSubValue1",
							},
						},
					},
					Global: []string{"globalKey1", "globalKey2"},
				}},
				{"repo1:production:group:service2", types.GroupConfig{
					Local: map[string]interface{}{
						"config": map[string]interface{}{
							"port":    float64(8080),
							"timeout": float64(30),
						},
					},
					Global: []string{"globalKey2", "globalKey3"},
				}},
				{"repo2:production:group:service3", types.GroupConfig{
					Local: map[string]interface{}{
						"serviceConfig": map[string]interface{}{
							"port":    float64(9090),
							"timeout": float64(45),
						},
					},
					Global: []string{"globalKey1", "globalKey3"},
				}},

				{"repo1:production:global:globalKey1", "globalValue1"},
				{"repo1:production:global:globalKey2", "globalValue2"},
				{"repo1:production:global:globalKey3", "globalValue3"},
				{"repo2:production:global:globalKey4", "globalValue4"},
				{"repo2:production:global:globalKey5", "globalValue5"},
				{"repo2:production:global:globalKey6", "globalValue6"},

				// develop
				{"repo1:develop:group:service1", types.GroupConfig{
					Local: map[string]interface{}{
						"nestedConfig": map[string]interface{}{
							"subKey1": "subValue1",
							"subKey2": map[string]interface{}{
								"subSubKey1": "subSubValue1",
							},
						},
					},
					Global: []string{"globalKey1", "globalKey2"},
				}},
				{"repo1:develop:group:service2", types.GroupConfig{
					Local: map[string]interface{}{
						"config": map[string]interface{}{
							"port":    float64(8080),
							"timeout": float64(30),
						},
					},
					Global: []string{"globalKey2", "globalKey3"},
				}},
				{"repo2:develop:group:service3", types.GroupConfig{
					Local: map[string]interface{}{
						"serviceConfig": map[string]interface{}{
							"port":    float64(9090),
							"timeout": float64(45),
						},
					},
					Global: []string{"globalKey1", "globalKey3"},
				}},

				{"repo1:develop:global:globalKey-dev-1", "globalValue-dev-1"},
				{"repo1:develop:global:globalKey-dev-2", "globalValue-dev-2"},
				{"repo1:develop:global:globalKey-dev-3", "globalValue-dev-3"},
				{"repo2:develop:global:globalKey-dev-4", "globalValue-dev-4"},
				{"repo2:develop:global:globalKey-dev-5", "globalValue-dev-5"},
				{"repo2:develop:global:globalKey-dev-6", "globalValue-dev-6"},

				// staging
				{"repo1:staging:group:service1", types.GroupConfig{
					Local: map[string]interface{}{
						"nestedConfig": map[string]interface{}{
							"subKey1": "subValue1",
							"subKey2": map[string]interface{}{
								"subSubKey1": "subSubValue1",
							},
						},
					},
					Global: []string{"globalKey1", "globalKey2"},
				}},
				{"repo1:staging:group:service2", types.GroupConfig{
					Local: map[string]interface{}{
						"config": map[string]interface{}{
							"port":    float64(8080),
							"timeout": float64(30),
						},
					},
					Global: []string{"globalKey2", "globalKey3"},
				}},
				{"repo2:staging:group:service3", types.GroupConfig{
					Local: map[string]interface{}{
						"serviceConfig": map[string]interface{}{
							"port":    float64(9090),
							"timeout": float64(45),
						},
					},
					Global: []string{"globalKey1", "globalKey3"},
				}},

				{"repo1:staging:global:globalKey-stg-1", "globalValue-stg-1"},
				{"repo1:staging:global:globalKey-stg-2", "globalValue-stg-2"},
				{"repo1:staging:global:globalKey-stg-3", "globalValue-stg-3"},
				{"repo2:staging:global:globalKey-stg-4", "globalValue-stg-4"},
				{"repo2:staging:global:globalKey-stg-5", "globalValue-stg-5"},
				{"repo2:staging:global:globalKey-stg-6", "globalValue-stg-6"},
			},
			expectedResult: map[string]interface{}{
				"service1": map[string]interface{}{
					"nestedConfig": map[string]interface{}{
						"subKey1": "subValue1",
						"subKey2": map[string]interface{}{
							"subSubKey1": "subSubValue1",
						},
					},
					"globalKey1": "globalValue1",
					"globalKey2": "globalValue2",
				},
				"service2": map[string]interface{}{
					"config": map[string]interface{}{
						"port":    float64(8080),
						"timeout": float64(30),
					},
					"globalKey2": "globalValue2",
					"globalKey3": "globalValue3",
				},
				"service3": map[string]interface{}{
					"serviceConfig": map[string]interface{}{
						"port":    float64(9090),
						"timeout": float64(45),
					},
					"globalKey1": "globalValue1",
					"globalKey3": "globalValue3",
				},
				"globalKey4": "globalValue4",
				"globalKey5": "globalValue5",
				"globalKey6": "globalValue6",
			},
			expectedErr: false,
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

				err = suite.client.Set(ctx, repository.ConfigurationPrefix+":"+data.key, value, 0).Err()
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

func (suite *RedisRepositorySuite) TestCloneConfig() {
	ctx := context.Background()
	t := suite.T()

	testCases := []struct {
		name             string
		prePopulate      map[string]interface{}
		cloneEnv         string
		newEnv           string
		changedGlobalKey map[string]interface{}
		expectedKeys     map[string]interface{}
		expectError      bool
		expectedAllKeys  []string
	}{
		{
			name: "Clone 'develop' to 'develop-clone' with multiple keys and replacing 'global-key-2'",
			prePopulate: map[string]interface{}{
				"repo1:develop:group:service-1": map[string]interface{}{
					"key1": "value1",
				},
				"repo2:develop:group:service-2": map[string]interface{}{
					"key1": "value1",
				},
				"repo1:develop:global:global-key-1": map[string]interface{}{
					"key2": "value2",
				},
				"repo2:develop:global:global-key-2": map[string]interface{}{
					"key2": "value2",
				},
			},
			cloneEnv: "develop",
			newEnv:   "develop-clone",
			changedGlobalKey: map[string]interface{}{
				"global-key-2": map[string]interface{}{
					"new": "new",
				},
			},
			expectedKeys: map[string]interface{}{
				"repo1:develop-clone:group:service-1": map[string]interface{}{
					"key1": "value1",
				},
				"repo2:develop-clone:group:service-2": map[string]interface{}{
					"key1": "value1",
				},
				"repo1:develop-clone:global:global-key-1": map[string]interface{}{
					"key2": "value2",
				},
				"repo2:develop-clone:global:global-key-2": map[string]interface{}{
					"new": "new",
				},
			},
			expectError: false,
			expectedAllKeys: []string{
				"repo1:develop:group:service-1",
				"repo2:develop:group:service-2",
				"repo1:develop:global:global-key-1",
				"repo2:develop:global:global-key-2",
				"repo1:develop-clone:group:service-1",
				"repo2:develop-clone:group:service-2",
				"repo1:develop-clone:global:global-key-1",
				"repo2:develop-clone:global:global-key-2",
			},
		},
		{
			name: "Clone 'develop' to 'develop-clone' with multiple keys and NOT replacing any global keys",
			prePopulate: map[string]interface{}{
				"repo1:develop:group:service-1": map[string]interface{}{
					"key1": "value1",
				},
				"repo2:develop:group:service-2": map[string]interface{}{
					"key1": "value1",
				},
				"repo1:develop:global:global-key-1": map[string]interface{}{
					"key2": "value2",
				},
				"repo2:develop:global:global-key-2": map[string]interface{}{
					"key2": "value2",
				},
			},
			cloneEnv: "develop",
			newEnv:   "develop-clone",
			expectedKeys: map[string]interface{}{
				"repo1:develop-clone:group:service-1": map[string]interface{}{
					"key1": "value1",
				},
				"repo2:develop-clone:group:service-2": map[string]interface{}{
					"key1": "value1",
				},
				"repo1:develop-clone:global:global-key-1": map[string]interface{}{
					"key2": "value2",
				},
				"repo2:develop-clone:global:global-key-2": map[string]interface{}{
					"key2": "value2",
				},
			},
			expectError: false,
			expectedAllKeys: []string{
				"repo1:develop:group:service-1",
				"repo2:develop:group:service-2",
				"repo1:develop:global:global-key-1",
				"repo2:develop:global:global-key-2",
				"repo1:develop-clone:group:service-1",
				"repo2:develop-clone:group:service-2",
				"repo1:develop-clone:global:global-key-1",
				"repo2:develop-clone:global:global-key-2",
			},
		},
		{
			name: "Comprehensive cloning with nested structures across repos and envs",
			prePopulate: map[string]interface{}{
				"repo1:develop:group:service-1": map[string]interface{}{
					"key1":     "value1",
					"arrayKey": []interface{}{"item1", map[string]interface{}{"itemKey": "itemValue"}},
				},
				"repo1:develop:global:global-key-1": map[string]interface{}{
					"globalNested": map[string]interface{}{"nestedKey": "nestedValue"},
				},
				"repo2:develop:group:service-2": map[string]interface{}{
					"key2": "value2",
				},
				"repo2:develop:global:global-key-2": "simpleGlobalValue",
				"repo3:release:group:service-3": map[string]interface{}{
					"key3": []interface{}{"releaseVal1", "releaseVal2"},
				},
				"repo3:release:global:global-key-3": map[string]interface{}{
					"releaseGlobalKey": "releaseGlobalValue",
				},
				"repo1:production:group:service-1": "prodValue1",
				"repo2:production:global:global-key-2": map[string]interface{}{
					"prodGlobalNested": map[string]interface{}{"prodNestedKey": "prodNestedValue"},
				},
			},
			cloneEnv: "develop",
			newEnv:   "develop-clone",
			changedGlobalKey: map[string]interface{}{
				"global-key-1": map[string]interface{}{
					"new": map[string]interface{}{"new": "new"},
				},
			},
			expectedKeys: map[string]interface{}{
				"repo1:develop-clone:group:service-1": map[string]interface{}{
					"key1":     "value1",
					"arrayKey": []interface{}{"item1", map[string]interface{}{"itemKey": "itemValue"}},
				},
				"repo1:develop-clone:global:global-key-1": map[string]interface{}{
					"new": map[string]interface{}{"new": "new"},
				},
				"repo2:develop-clone:group:service-2": map[string]interface{}{
					"key2": "value2",
				},
				"repo2:develop-clone:global:global-key-2": "simpleGlobalValue",
				"repo1:production:group:service-1":        "prodValue1",
				"repo2:production:global:global-key-2": map[string]interface{}{
					"prodGlobalNested": map[string]interface{}{"prodNestedKey": "prodNestedValue"},
				},
			},
			expectError: false,
			expectedAllKeys: []string{
				"repo1:develop:group:service-1",
				"repo1:develop:global:global-key-1",
				"repo2:develop:group:service-2",
				"repo2:develop:global:global-key-2",
				"repo3:release:group:service-3",
				"repo3:release:global:global-key-3",
				"repo1:develop-clone:group:service-1",
				"repo1:develop-clone:global:global-key-1",
				"repo2:develop-clone:group:service-2",
				"repo2:develop-clone:global:global-key-2",
				"repo1:production:group:service-1",
				"repo2:production:global:global-key-2",
			},
		},
		{
			name: "Clone with all pre-populated keys having changed values",
			prePopulate: map[string]interface{}{
				"r1:release:group:s1": map[string]interface{}{
					"key1": "value1",
				},
				"r1:release:global:gk1": map[string]interface{}{
					"key2": "value2",
				},
				"r2:release:group:s2": map[string]interface{}{
					"key3": "value3",
				},
				"r2:release:global:gk2": map[string]interface{}{
					"key4": "value4",
				},
				"r3:release:group:s3": map[string]interface{}{
					"key5": "value5",
				},
				"r3:release:global:gk3": map[string]interface{}{
					"key6": "value6",
				},
			},
			cloneEnv: "release",
			newEnv:   "clone",
			changedGlobalKey: map[string]interface{}{
				"gk1": map[string]interface{}{
					"key2": "changedValue2",
				},
				"gk2": map[string]interface{}{
					"key4": "changedValue4",
				},
				"gk3": map[string]interface{}{
					"key6": "changedValue6",
				},
			},
			expectedKeys: map[string]interface{}{
				"r1:clone:group:s1": map[string]interface{}{
					"key1": "value1",
				},
				"r1:clone:global:gk1": map[string]interface{}{
					"key2": "changedValue2",
				},
				"r2:clone:group:s2": map[string]interface{}{
					"key3": "value3",
				},
				"r2:clone:global:gk2": map[string]interface{}{
					"key4": "changedValue4",
				},
				"r3:clone:group:s3": map[string]interface{}{
					"key5": "value5",
				},
				"r3:clone:global:gk3": map[string]interface{}{
					"key6": "changedValue6",
				},
			},
			expectError: false,
			expectedAllKeys: []string{
				"r1:release:group:s1",
				"r1:release:global:gk1",
				"r2:release:group:s2",
				"r2:release:global:gk2",
				"r3:release:group:s3",
				"r3:release:global:gk3",
				"r1:clone:group:s1",
				"r1:clone:global:gk1",
				"r2:clone:group:s2",
				"r2:clone:global:gk2",
				"r3:clone:group:s3",
				"r3:clone:global:gk3",
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.BeforeTest(tc.name)

			// Setup initial keys in Redis
			for k, v := range tc.prePopulate {
				value, err := json.Marshal(v)
				suite.NoError(err, "Marshalling pre-populated keys")

				fullKey := repository.ConfigurationPrefix + ":" + k

				err = suite.client.Set(ctx, fullKey, value, 0).Err()
				assert.NoError(t, err, "Setting up keys for test case")
			}

			// Execute CloneConfig
			err := suite.repository.CloneConfig(ctx, tc.cloneEnv, tc.newEnv, tc.changedGlobalKey)

			if tc.expectError {
				assert.Error(t, err, "Expected an error")
			} else {
				assert.NoError(t, err, "Expected no error")
			}

			// Verify expected keys are created with correct values
			for expectedKey, expectedValue := range tc.expectedKeys {
				fullExpectedKey := repository.ConfigurationPrefix + ":" + expectedKey

				var actualValue interface{}
				val, err := suite.client.Get(ctx, fullExpectedKey).Result()
				assert.NoError(suite.T(), err)

				err = json.Unmarshal([]byte(val), &actualValue)
				assert.NoError(suite.T(), err)

				assert.NoError(t, err, "Fetching cloned key")
				assert.Equal(t, expectedValue, actualValue, fmt.Sprintf("Value mismatch for key %s", fullExpectedKey))
			}

			// Verify that original keys has not been changed
			for originalKey, originalValue := range tc.prePopulate {
				fullOriginalKey := repository.ConfigurationPrefix + ":" + originalKey
				var actualValue interface{}
				val, err := suite.client.Get(ctx, fullOriginalKey).Result()
				assert.NoError(suite.T(), err)

				err = json.Unmarshal([]byte(val), &actualValue)
				assert.NoError(suite.T(), err)

				assert.NoError(t, err, "Fetching cloned key")
				assert.Equal(t, originalValue, actualValue, fmt.Sprintf("Value mismatch for key %s", fullOriginalKey))
			}

			// Verify that the the existing keys are the expected
			allKeys, err := suite.client.Keys(ctx, "*").Result()
			assert.NoError(t, err, "Fetching all keys")
			expectedFullKeys := []string{}

			for _, k := range tc.expectedAllKeys {
				fullExpectedKey := repository.ConfigurationPrefix + ":" + k
				expectedFullKeys = append(expectedFullKeys, fullExpectedKey)
			}

			assert.ElementsMatch(t, allKeys, expectedFullKeys, fmt.Sprintf("Value mismatch for all generated keys %v", tc.expectedAllKeys))
		})
	}
}

func (suite *RedisRepositorySuite) TestAddEnv() {
	ctx := context.Background()

	testCases := []struct {
		name           string
		envName        string
		params         repository.EnvParams
		prePopulate    map[string]interface{}
		expectedParams map[string]string
		expectError    bool
		expectedError  error
	}{
		{
			name:    "Add environment metadata successfully",
			envName: "test-env",
			params: repository.EnvParams{
				Name:     "test-name",
				Version:  "test-version",
				Clone:    true,
				Original: "test-original",
			},
			prePopulate: map[string]interface{}{},
			expectedParams: map[string]string{
				"name":     "test-name",
				"version":  "test-version",
				"clone":    "1",
				"original": "test-original",
			},
			expectError:   false,
			expectedError: nil,
		},
		{
			name:    "Add environment metadata successfully",
			envName: "test-env",
			params: repository.EnvParams{
				Name:     "test-name",
				Version:  "test-version",
				Clone:    true,
				Original: "test-original",
			},
			prePopulate: map[string]interface{}{},
			expectedParams: map[string]string{
				"name":     "test-name",
				"version":  "test-version",
				"clone":    "1",
				"original": "test-original",
			},
			expectError:   false,
			expectedError: nil,
		},
		{
			name:    "Add environment metadata with empty name",
			envName: "",
			params: repository.EnvParams{
				Name:     "",
				Version:  "test-version",
				Clone:    false,
				Original: "test-original",
			},
			prePopulate:    map[string]interface{}{},
			expectedParams: map[string]string{},
			expectError:    true,
			expectedError:  errors.New("environment name cannot be empty"),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.BeforeTest(tc.name)

			for k, v := range tc.prePopulate {
				value, err := json.Marshal(v)
				suite.NoError(err, "Marshalling pre-populated keys")

				fullKey := suite.repository.GetEnvKey(k)

				err = suite.client.HMSet(ctx, fullKey, value, 0).Err()
				suite.NoError(err, "Setting up keys for test case")
			}

			err := suite.repository.AddEnv(ctx, tc.envName, tc.params)
			if tc.expectError {
				suite.Error(err, "Expected an error")
				suite.Equal(tc.expectedError, err, "Error mismatch")
			} else {
				suite.NoError(err, "Expected no error")
			}

			fullExpectedKey := suite.repository.GetEnvKey(tc.envName)

			actualParams, err := suite.client.HGetAll(ctx, suite.repository.GetEnvKey(tc.envName)).Result()
			suite.NoError(err)

			suite.Equal(tc.expectedParams, actualParams, fmt.Sprintf("Value mismatch for key %s", fullExpectedKey))
		})
	}
}

func (suite *RedisRepositorySuite) TestDeleteEnv() {
	ctx := context.Background()

	testCases := []struct {
		name        string
		envName     string
		prePopulate map[string]string
		expectError bool
		expectedErr error
	}{
		{
			name:        "Remove environment metadata successfully",
			envName:     "test-env",
			prePopulate: map[string]string{"test-env:name": "test-name", "test-env:version": "test-version"},
			expectError: false,
			expectedErr: nil,
		},
		{
			name:        "Remove non-existing environment metadata",
			envName:     "non-existing-env",
			prePopulate: map[string]string{},
			expectError: false,
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.BeforeTest(tc.name)

			for k, v := range tc.prePopulate {
				value, err := json.Marshal(v)
				suite.NoError(err, "Marshalling pre-populated keys")

				fullKey := suite.repository.GetEnvKey(k)

				err = suite.client.HMSet(ctx, fullKey, value, 0).Err()
				suite.NoError(err, "Setting up keys for test case")
			}

			err := suite.repository.DeleteEnv(ctx, tc.envName)

			if tc.expectError {
				suite.Error(err, "Expected an error")
				suite.Equal(tc.expectedErr, err, "Error mismatch")
			} else {
				suite.NoError(err, "Expected no error")
			}

			// Verify that the key is removed from Redis
			exists, err := suite.client.Exists(ctx, suite.repository.GetEnvKey(tc.envName)).Result()
			suite.NoError(err)
			suite.Equal(int64(0), exists, "Expected key to be removed from Redis")
		})
	}
}

func (suite *RedisRepositorySuite) TestGetEnvOriginal() {
	ctx := context.Background()

	testCases := []struct {
		name        string
		envName     string
		prePopulate map[string]string
		expectError bool
		expectedOk  bool
		expectedErr error
		expectedVal string
	}{
		{
			name:        "Get original environment value successfully",
			envName:     "test-env",
			prePopulate: map[string]string{"name": "test-name", "version": "test-version", "original": "test-original"},
			expectError: false,
			expectedOk:  true,
			expectedErr: nil,
			expectedVal: "test-original",
		},
		{
			name:        "Get original value of non-existing environment",
			envName:     "non-existing-env",
			prePopulate: nil,
			expectError: false,
			expectedOk:  false,
			expectedErr: nil,
			expectedVal: "",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.BeforeTest(tc.name)

			if tc.prePopulate != nil {
				err := suite.client.HMSet(ctx, suite.repository.GetEnvKey(tc.envName), tc.prePopulate).Err()
				suite.NoError(err, "Setting up keys for test case")
			}

			originalVal, ok, err := suite.repository.GetEnvOriginal(ctx, tc.envName)
			suite.Equal(tc.expectedOk, ok, "Ok mismatch")

			if tc.expectError {
				suite.Error(err, "Expected an error")
				suite.Equal(tc.expectedErr, err, "Error mismatch")
			} else {
				suite.NoError(err, "Expected no error")
				suite.Equal(tc.expectedVal, originalVal, "Original value mismatch")
			}
		})
	}
}

func (suite *RedisRepositorySuite) TestSetEnvVersion() {
	ctx := context.Background()

	testCases := []struct {
		name            string
		envName         string
		version         string
		prePopulate     map[string]string
		expectError     bool
		expectedErr     error
		expectedVersion string
	}{
		{
			name:            "Set environment version successfully",
			envName:         "test-env",
			version:         "1.0.0",
			prePopulate:     map[string]string{"name": "test-name", "original": "test-original"},
			expectError:     false,
			expectedErr:     nil,
			expectedVersion: "1.0.0",
		},
		{
			name:            "Set environment version for non-existing environment",
			envName:         "non-existing-env",
			version:         "1.0.0",
			prePopulate:     nil,
			expectError:     false,
			expectedErr:     nil,
			expectedVersion: "1.0.0",
		},
		{
			name:            "Set environment version with empty environment name",
			envName:         "",
			version:         "1.0.0",
			prePopulate:     nil,
			expectError:     true,
			expectedErr:     errors.New("environment name cannot be empty"),
			expectedVersion: "",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.BeforeTest(tc.name)

			if tc.prePopulate != nil {
				err := suite.client.HMSet(ctx, suite.repository.GetEnvKey(tc.envName), tc.prePopulate).Err()
				suite.NoError(err, "Setting up keys for test case")
			}

			err := suite.repository.SetEnvVersion(ctx, tc.envName, tc.version)
			if tc.expectError {
				suite.Error(err, "Expected an error")
				suite.Equal(tc.expectedErr, err, "Error mismatch")
			} else {
				suite.NoError(err, "Expected no error")

				actualVersion, err := suite.client.HGet(ctx, suite.repository.GetEnvKey(tc.envName), "version").Result()
				suite.NoError(err, "Failed to get version from Redis")
				suite.Equal(tc.expectedVersion, actualVersion, "Version mismatch")
			}
		})
	}
}

func (suite *RedisRepositorySuite) TestGetEnvParams() {
	ctx := context.Background()

	testCases := []struct {
		name           string
		envName        string
		prePopulate    map[string]string
		expectError    bool
		expectedErr    error
		expectedParams repository.EnvParams
	}{
		{
			name:           "Get environment params successfully",
			envName:        "test-env",
			prePopulate:    map[string]string{"version": "1.0.0", "clone": "1", "original": "test-original", "name": "test-env"},
			expectError:    false,
			expectedErr:    nil,
			expectedParams: repository.EnvParams{Name: "test-env", Version: "1.0.0", Clone: true, Original: "test-original"},
		},
		{
			name:           "Get environment params with empty key",
			envName:        "",
			prePopulate:    nil,
			expectError:    true,
			expectedErr:    errors.New("environment name cannot be empty"),
			expectedParams: repository.EnvParams{},
		},
		{
			name:           "Get environment params with non-existing key",
			envName:        "non-existing-key",
			prePopulate:    nil,
			expectError:    true,
			expectedErr:    repository.EnvNotFoundError{Key: "non-existing-key"},
			expectedParams: repository.EnvParams{},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.BeforeTest(tc.name)

			if tc.prePopulate != nil {
				err := suite.client.HMSet(ctx, suite.repository.GetEnvKey(tc.envName), tc.prePopulate).Err()
				suite.NoError(err, "Setting up keys for test case")
			}

			params, err := suite.repository.GetEnvParams(ctx, tc.envName)
			if tc.expectError {
				suite.Error(err, "Expected an error")
				suite.Equal(tc.expectedErr, err, "Error mismatch")
			} else {
				suite.NoError(err, "Expected no error")
				suite.Equal(tc.expectedParams, params, "Params mismatch")
			}
		})
	}
}

func (suite *RedisRepositorySuite) TestGetAllEnvs() {
	ctx := context.Background()

	testCases := []struct {
		name           string
		keys           []string
		prePopulate    map[string]map[string]string
		expectedParams []repository.EnvParams
		expectError    bool
		expectedErr    error
	}{
		{
			name: "Get all environments successfully where there are only no cloned environments",
			prePopulate: map[string]map[string]string{
				"develop":    {"name": "develop", "version": "1.0.0", "clone": "0"},
				"release":    {"name": "release", "version": "2.0.0", "clone": "0"},
				"production": {"name": "production", "version": "3.0.0", "clone": "0"},
			},
			expectedParams: []repository.EnvParams{
				{Name: "develop", Version: "1.0.0", Clone: false},
				{Name: "release", Version: "2.0.0", Clone: false},
				{Name: "production", Version: "3.0.0", Clone: false},
			},
			expectError: false,
			expectedErr: nil,
		},
		{
			name: "Get all environments successfully where there are cloned and not environments",
			prePopulate: map[string]map[string]string{
				"develop":         {"name": "develop", "version": "2.0.0", "clone": "0"},
				"develop-clone-1": {"name": "develop-clone-1", "version": "1.0.0", "clone": "1", "original": "develop"},
				"develop-clone-2": {"name": "develop-clone-2", "version": "2.0.0", "clone": "1", "original": "develop"},
				"release":         {"name": "release", "version": "2.0.0", "clone": "0"},
				"production":      {"name": "production", "version": "3.0.0", "clone": "0"},
			},
			expectedParams: []repository.EnvParams{
				{Name: "develop", Version: "2.0.0", Clone: false},
				{Name: "develop-clone-1", Version: "1.0.0", Clone: true, Original: "develop"},
				{Name: "develop-clone-2", Version: "2.0.0", Clone: true, Original: "develop"},
				{Name: "release", Version: "2.0.0", Clone: false},
				{Name: "production", Version: "3.0.0", Clone: false},
			},
			expectError: false,
			expectedErr: nil,
		},
		{
			name:           "No environments found",
			prePopulate:    map[string]map[string]string{},
			expectedParams: []repository.EnvParams{},
			expectError:    false,
			expectedErr:    nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.BeforeTest(tc.name)

			if tc.prePopulate != nil {
				for envName, m := range tc.prePopulate {
					err := suite.client.HMSet(ctx, suite.repository.GetEnvKey(envName), m).Err()
					suite.NoError(err, "Setting up keys for test case")
				}
			}

			params, err := suite.repository.GetAllEnvs(ctx)
			if tc.expectError {
				suite.Error(err, "Expected an error")
				suite.Equal(tc.expectedErr, err, "Error mismatch")
			} else {
				suite.NoError(err, "Expected no error")
				suite.ElementsMatch(tc.expectedParams, params, "Params mismatch")
			}
		})
	}
}

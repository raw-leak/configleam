package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/raw-leak/configleam/internal/app/configuration/repository"
	"github.com/raw-leak/configleam/internal/app/configuration/types"
	"github.com/raw-leak/configleam/internal/pkg/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type EtcdRepositorySuite struct {
	suite.Suite
	repository *repository.EtcdRepository
	client     *clientv3.Client
	keys       repository.EtcdKeys
}

func TestEtcdRepositorySuite(t *testing.T) {
	suite.Run(t, new(EtcdRepositorySuite))
}

func (suite *EtcdRepositorySuite) SetupSuite() {
	addrs := "http://localhost:8079"

	client, err := clientv3.New(clientv3.Config{
		Endpoints: []string{addrs},
	})
	suite.NoErrorf(err, "error connecting to etcd server %s", addrs)

	suite.keys = repository.EtcdKeys{}
	suite.client = client
	suite.repository = repository.NewEtcdRepository(&etcd.Etcd{Client: suite.client})
}

func (suite *EtcdRepositorySuite) TearDownSuite() {
	suite.client.Close()
}

func (suite *EtcdRepositorySuite) getKeysWithPrefix(ctx context.Context, prefix string) ([]string, error) {
	var keys []string

	rangeEnd := clientv3.GetPrefixRangeEnd(prefix)
	resp, err := suite.client.Get(ctx, prefix, clientv3.WithRange(rangeEnd))
	if err != nil {
		return nil, err
	}

	for _, kv := range resp.Kvs {
		keys = append(keys, string(kv.Key))
	}

	return keys, nil
}

func (suite *EtcdRepositorySuite) getOneKey(ctx context.Context, key string) ([]byte, error) {
	res, err := suite.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if len(res.Kvs) > 1 {
		return nil, fmt.Errorf("error fetching more than one key %s: %v", key, err)
	}

	if len(res.Kvs) == 0 {
		return nil, nil
	}

	return res.Kvs[0].Value, nil
}

func (suite *EtcdRepositorySuite) BeforeTest(testName string) {
	_, err := suite.client.Delete(context.Background(), "", clientv3.WithPrefix())
	assert.NoErrorf(suite.T(), err, "Deleting all data from ETCD before each test within the test: %s", testName)
}

func (suite *EtcdRepositorySuite) TestUpsertConfig() {
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
					"app1": {
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
					"nonexistent": {
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
					"service1": {
						Local: map[string]interface{}{
							"servicePort": 8080,
							"servicePath": "/api",
						},
						Global: []string{"globalKey1", "globalKey2"},
					},
					"service2": {
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

			ctx := context.Background()

			err := suite.repository.UpsertConfig(ctx, tc.repo, tc.env, tc.config)

			if tc.expectedErr {
				suite.Error(err)

				keys, err := suite.getKeysWithPrefix(ctx, repository.ConfigurationPrefix+":"+tc.repo+":"+tc.env+":")
				suite.NoError(err)
				suite.Equal(0, len(keys))
			} else {
				suite.NoError(err)

				// check globals
				for _, g := range tc.expectedGlobals {
					for key, expectedValue := range g {

						val, err := suite.getOneKey(ctx, repository.ConfigurationPrefix+":"+key)
						suite.NoError(err)

						var actualValue interface{}
						err = json.Unmarshal(val, &actualValue)
						suite.NoError(err)

						suite.Equal(expectedValue, actualValue)
					}
				}

				// check groups
				for _, g := range tc.expectedGlobals {
					for key, expectedValue := range g {
						val, err := suite.getOneKey(ctx, repository.ConfigurationPrefix+":"+key)
						suite.NoError(err)

						var actualValue interface{}
						err = json.Unmarshal(val, &actualValue)
						suite.NoError(err)

						suite.Equal(expectedValue, actualValue)
					}
				}

				// check globals
				for _, g := range tc.expectedGroups {
					for key, expectedValue := range g {
						val, err := suite.getOneKey(ctx, repository.ConfigurationPrefix+":"+key)
						suite.NoError(err)

						var actualValue interface{}
						err = json.Unmarshal([]byte(val), &actualValue)
						suite.NoError(err)

						suite.Equal(expectedValue, actualValue)
					}
				}

				// ensure only expected keys exist
				keys, err := suite.getKeysWithPrefix(ctx, repository.ConfigurationPrefix+":"+tc.repo+":"+tc.env)
				suite.NoError(err)
				suite.Equal(len(tc.expectedKeys), len(keys))

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

func (suite *EtcdRepositorySuite) TestReadConfig() {
	type prePopulateData struct {
		key   string
		value interface{}
	}

	type testCase struct {
		name           string
		repo           string
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
			repo:       "test-repo",
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
			repo:       "staging-repo",
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
			repo:       "repo1",
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
				{"repo1:production:group:service3", types.GroupConfig{
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
				{"repo1:production:global:globalKey4", "globalValue4"},
				{"repo1:production:global:globalKey5", "globalValue5"},
				{"repo1:production:global:globalKey6", "globalValue6"},

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
				{"repo1:develop:group:service3", types.GroupConfig{
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
				{"repo1:develop:global:globalKey-dev-4", "globalValue-dev-4"},
				{"repo1:develop:global:globalKey-dev-5", "globalValue-dev-5"},
				{"repo1:develop:global:globalKey-dev-6", "globalValue-dev-6"},

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
				{"repo1:staging:group:service3", types.GroupConfig{
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
				{"repo1:staging:global:globalKey-stg-4", "globalValue-stg-4"},
				{"repo1:staging:global:globalKey-stg-5", "globalValue-stg-5"},
				{"repo1:staging:global:globalKey-stg-6", "globalValue-stg-6"},
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

			for _, data := range tc.prePopulate {
				value, err := json.Marshal(data.value)
				suite.Require().NoError(err)

				_, err = suite.client.Put(ctx, repository.ConfigurationPrefix+":"+data.key, string(value))
				suite.Require().NoError(err)
			}

			result, err := suite.repository.ReadConfig(ctx, tc.repo, tc.env, tc.groups, tc.globalKeys)

			if tc.expectedErr {
				suite.Assert().Error(err)
			} else {
				suite.Assert().NoError(err)
				suite.Assert().Equal(tc.expectedResult, result)
			}
		})
	}
}

func (suite *EtcdRepositorySuite) TestCloneConfig() {
	ctx := context.Background()

	testCases := []struct {
		name             string
		repo             string
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
			repo: "repo1",
			prePopulate: map[string]interface{}{
				"repo1:develop:group:service-1": map[string]interface{}{
					"key1": "value1",
				},
				"repo1:develop:group:service-2": map[string]interface{}{
					"key1": "value1",
				},
				"repo1:develop:global:global-key-1": map[string]interface{}{
					"key2": "value2",
				},
				"repo1:develop:global:global-key-2": map[string]interface{}{
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
				"repo1:develop-clone:group:service-2": map[string]interface{}{
					"key1": "value1",
				},
				"repo1:develop-clone:global:global-key-1": map[string]interface{}{
					"key2": "value2",
				},
				"repo1:develop-clone:global:global-key-2": map[string]interface{}{
					"new": "new",
				},
			},
			expectError: false,
			expectedAllKeys: []string{
				"repo1:develop:group:service-1",
				"repo1:develop:group:service-2",
				"repo1:develop:global:global-key-1",
				"repo1:develop:global:global-key-2",
				"repo1:develop-clone:group:service-1",
				"repo1:develop-clone:group:service-2",
				"repo1:develop-clone:global:global-key-1",
				"repo1:develop-clone:global:global-key-2",
			},
		},
		{
			name: "Clone 'develop' to 'develop-clone' with multiple keys and NOT replacing any global keys",
			repo: "repo1",
			prePopulate: map[string]interface{}{
				"repo1:develop:group:service-1": map[string]interface{}{
					"key1": "value1",
				},
				"repo1:develop:group:service-2": map[string]interface{}{
					"key1": "value1",
				},
				"repo1:develop:global:global-key-1": map[string]interface{}{
					"key2": "value2",
				},
				"repo1:develop:global:global-key-2": map[string]interface{}{
					"key2": "value2",
				},
			},
			cloneEnv: "develop",
			newEnv:   "develop-clone",
			expectedKeys: map[string]interface{}{
				"repo1:develop-clone:group:service-1": map[string]interface{}{
					"key1": "value1",
				},
				"repo1:develop-clone:group:service-2": map[string]interface{}{
					"key1": "value1",
				},
				"repo1:develop-clone:global:global-key-1": map[string]interface{}{
					"key2": "value2",
				},
				"repo1:develop-clone:global:global-key-2": map[string]interface{}{
					"key2": "value2",
				},
			},
			expectError: false,
			expectedAllKeys: []string{
				"repo1:develop:group:service-1",
				"repo1:develop:group:service-2",
				"repo1:develop:global:global-key-1",
				"repo1:develop:global:global-key-2",
				"repo1:develop-clone:group:service-1",
				"repo1:develop-clone:group:service-2",
				"repo1:develop-clone:global:global-key-1",
				"repo1:develop-clone:global:global-key-2",
			},
		},
		{
			name: "Comprehensive cloning with nested structures across repos and envs",
			repo: "repo1",
			prePopulate: map[string]interface{}{
				"repo1:develop:group:service-1": map[string]interface{}{
					"key1":     "value1",
					"arrayKey": []interface{}{"item1", map[string]interface{}{"itemKey": "itemValue"}},
				},
				"repo1:develop:global:global-key-1": map[string]interface{}{
					"globalNested": map[string]interface{}{"nestedKey": "nestedValue"},
				},
				"repo1:develop:group:service-2": map[string]interface{}{
					"key2": "value2",
				},
				"repo1:develop:global:global-key-2": "simpleGlobalValue",
				"repo3:release:group:service-3": map[string]interface{}{
					"key3": []interface{}{"releaseVal1", "releaseVal2"},
				},
				"repo3:release:global:global-key-3": map[string]interface{}{
					"releaseGlobalKey": "releaseGlobalValue",
				},
				"repo1:production:group:service-1": "prodValue1",
				"repo1:production:global:global-key-2": map[string]interface{}{
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
				"repo1:develop-clone:group:service-2": map[string]interface{}{
					"key2": "value2",
				},
				"repo1:develop-clone:global:global-key-2": "simpleGlobalValue",
				"repo1:production:group:service-1":        "prodValue1",
				"repo1:production:global:global-key-2": map[string]interface{}{
					"prodGlobalNested": map[string]interface{}{"prodNestedKey": "prodNestedValue"},
				},
			},
			expectError: false,
			expectedAllKeys: []string{
				"repo1:develop:group:service-1",
				"repo1:develop:global:global-key-1",
				"repo1:develop:group:service-2",
				"repo1:develop:global:global-key-2",
				"repo3:release:group:service-3",
				"repo3:release:global:global-key-3",
				"repo1:develop-clone:group:service-1",
				"repo1:develop-clone:global:global-key-1",
				"repo1:develop-clone:group:service-2",
				"repo1:develop-clone:global:global-key-2",
				"repo1:production:group:service-1",
				"repo1:production:global:global-key-2",
			},
		},
		{
			name: "Clone with all pre-populated keys having changed values",
			repo: "r1",
			prePopulate: map[string]interface{}{
				"r1:release:group:s1": map[string]interface{}{
					"key1": "value1",
				},
				"r1:release:global:gk1": map[string]interface{}{
					"key2": "value2",
				},
				"r1:release:group:s2": map[string]interface{}{
					"key3": "value3",
				},
				"r1:release:global:gk2": map[string]interface{}{
					"key4": "value4",
				},
				"r1:release:group:s3": map[string]interface{}{
					"key5": "value5",
				},
				"r1:release:global:gk3": map[string]interface{}{
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
				"r1:clone:group:s2": map[string]interface{}{
					"key3": "value3",
				},
				"r1:clone:global:gk2": map[string]interface{}{
					"key4": "changedValue4",
				},
				"r1:clone:group:s3": map[string]interface{}{
					"key5": "value5",
				},
				"r1:clone:global:gk3": map[string]interface{}{
					"key6": "changedValue6",
				},
			},
			expectError: false,
			expectedAllKeys: []string{
				"r1:release:group:s1",
				"r1:release:global:gk1",
				"r1:release:group:s2",
				"r1:release:global:gk2",
				"r1:release:group:s3",
				"r1:release:global:gk3",
				"r1:clone:group:s1",
				"r1:clone:global:gk1",
				"r1:clone:group:s2",
				"r1:clone:global:gk2",
				"r1:clone:group:s3",
				"r1:clone:global:gk3",
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

				_, err = suite.client.Put(ctx, fullKey, string(value))
				suite.NoError(err, "Setting up keys for test case")
			}

			// Execute CloneConfig
			err := suite.repository.CloneConfig(ctx, tc.repo, tc.cloneEnv, tc.newEnv, tc.changedGlobalKey)

			if tc.expectError {
				suite.Error(err, "Expected an error")
			} else {
				suite.NoError(err, "Expected no error")
			}

			// Verify expected keys are created with correct values
			for expectedKey, expectedValue := range tc.expectedKeys {
				fullExpectedKey := repository.ConfigurationPrefix + ":" + expectedKey

				var actualValue interface{}
				res, err := suite.client.Get(ctx, fullExpectedKey)
				suite.NoError(err)

				val := res.Kvs[0].Value

				err = json.Unmarshal(val, &actualValue)
				suite.NoError(err)

				suite.NoError(err, "Fetching cloned key")
				suite.Equal(expectedValue, actualValue, fmt.Sprintf("Value mismatch for key %s", fullExpectedKey))
			}

			// Verify that original keys has not been changed
			for originalKey, originalValue := range tc.prePopulate {
				fullOriginalKey := repository.ConfigurationPrefix + ":" + originalKey
				var actualValue interface{}
				res, err := suite.client.Get(ctx, fullOriginalKey)
				suite.NoError(err)
				val := res.Kvs[0].Value

				err = json.Unmarshal(val, &actualValue)
				suite.NoError(err)

				suite.NoError(err, "Fetching cloned key")
				suite.Equal(originalValue, actualValue, fmt.Sprintf("Value mismatch for key %s", fullOriginalKey))
			}

			// Verify that the the existing keys are the expected
			res, err := suite.client.Get(ctx, "\x00", clientv3.WithFromKey(), clientv3.WithKeysOnly())
			suite.NoError(err, "Fetching all keys")

			allKeys := []string{}
			expectedFullKeys := []string{}

			for _, k := range res.Kvs {
				allKeys = append(allKeys, string(k.Key))
			}

			for _, k := range tc.expectedAllKeys {
				fullExpectedKey := repository.ConfigurationPrefix + ":" + k
				expectedFullKeys = append(expectedFullKeys, fullExpectedKey)
			}

			suite.ElementsMatch(allKeys, expectedFullKeys, fmt.Sprintf("Value mismatch for all generated keys %v", tc.expectedAllKeys))
		})
	}
}

func (suite *EtcdRepositorySuite) TestAddEnv() {
	ctx := context.Background()

	testCases := []struct {
		name           string
		envName        string
		params         repository.EnvParams
		expectedParams repository.EnvParams
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
			expectedParams: repository.EnvParams{
				Name:     "test-name",
				Version:  "test-version",
				Clone:    true,
				Original: "test-original",
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
			expectedParams: repository.EnvParams{
				Name:     "test-name",
				Version:  "test-version",
				Clone:    true,
				Original: "test-original",
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
			expectedParams: repository.EnvParams{},
			expectError:    true,
			expectedError:  errors.New("environment name cannot be empty"),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.BeforeTest(tc.name)

			err := suite.repository.AddEnv(ctx, tc.envName, tc.params)
			if tc.expectError {
				suite.Error(err, "Expected an error")
				suite.Equal(tc.expectedError, err, "Error mismatch")
			} else {
				suite.NoError(err, "Expected no error")

				fullExpectedKey := suite.keys.GetEnvKey(tc.envName)

				res, err := suite.client.Get(ctx, suite.keys.GetEnvKey(tc.envName))
				suite.NoError(err)

				var actualParams repository.EnvParams
				err = json.Unmarshal(res.Kvs[0].Value, &actualParams)
				suite.NoError(err)

				suite.Equal(tc.expectedParams, actualParams, fmt.Sprintf("Value mismatch for key %s", fullExpectedKey))
			}
		})
	}
}

func (suite *EtcdRepositorySuite) TestDeleteEnv() {
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

				fullKey := suite.keys.GetEnvKey(k)

				_, err = suite.client.Put(ctx, fullKey, string(value))
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
			res, err := suite.client.Get(ctx, suite.keys.GetEnvKey(tc.envName))
			suite.NoError(err)

			suite.Equal(int64(0), res.Count, "Expected key to be removed from Redis")
		})
	}
}

func (suite *EtcdRepositorySuite) TestGetEnvOriginal() {
	ctx := context.Background()

	testCases := []struct {
		name        string
		envName     string
		prePopulate *repository.EnvParams
		expectError bool
		expectedOk  bool
		expectedErr error
		expectedVal string
	}{
		{
			name:        "Get original environment value successfully",
			envName:     "test-env",
			prePopulate: &repository.EnvParams{Name: "test-name", Version: "test-version", Original: "test-original"},
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
				jsonData, err := json.Marshal(*tc.prePopulate)
				suite.NoError(err, "Setting up keys for test case")

				_, err = suite.client.Put(ctx, suite.keys.GetEnvKey(tc.envName), string(jsonData))
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

func (suite *EtcdRepositorySuite) TestSetEnvVersion() {
	ctx := context.Background()

	testCases := []struct {
		name            string
		envName         string
		version         string
		prePopulate     *repository.EnvParams
		expectError     bool
		expectedErr     string
		expectedVersion string
	}{
		{
			name:            "Set environment version successfully",
			envName:         "test-env",
			version:         "2.0.0",
			prePopulate:     &repository.EnvParams{Name: "test-name", Original: "test-original", Version: "1.0.0"},
			expectError:     false,
			expectedVersion: "2.0.0",
		},
		{
			name:        "Set environment version for non-existing environment",
			envName:     "non-existing-env",
			version:     "1.0.0",
			prePopulate: nil,
			expectError: true,
			expectedErr: "env 'non-existing-env' not found",
		},
		{
			name:        "Set environment version with empty environment name",
			envName:     "",
			version:     "1.0.0",
			prePopulate: nil,
			expectError: true,
			expectedErr: "environment name cannot be empty",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.BeforeTest(tc.name)

			if tc.prePopulate != nil {
				value, err := json.Marshal(*tc.prePopulate)
				suite.NoError(err)

				_, err = suite.client.Put(ctx, suite.keys.GetEnvKey(tc.envName), string(value))
				suite.NoError(err)
			}

			err := suite.repository.SetEnvVersion(ctx, tc.envName, tc.version)
			if tc.expectError {
				suite.Error(err)
				suite.Equal(tc.expectedErr, err.Error())
			} else {
				suite.NoError(err)

				res, err := suite.client.Get(ctx, suite.keys.GetEnvKey(tc.envName))
				suite.NoError(err)

				var newParams repository.EnvParams
				err = json.Unmarshal(res.Kvs[0].Value, &newParams)
				suite.NoError(err)

				suite.Equal(tc.expectedVersion, newParams.Version)
			}
		})
	}
}

func (suite *EtcdRepositorySuite) TestGetAllEnvs() {
	ctx := context.Background()

	testCases := []struct {
		name           string
		keys           []string
		prePopulate    map[string]repository.EnvParams
		expectedParams []repository.EnvParams
		expectError    bool
		expectedErr    error
	}{
		{
			name: "Get all environments successfully where there are only no cloned environments",
			prePopulate: map[string]repository.EnvParams{
				"develop":    {Name: "develop", Version: "1.0.0", Clone: false},
				"release":    {Name: "release", Version: "2.0.0", Clone: false},
				"production": {Name: "production", Version: "3.0.0", Clone: false},
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
			prePopulate: map[string]repository.EnvParams{
				"develop":         {Name: "develop", Version: "2.0.0", Clone: false},
				"develop-clone-1": {Name: "develop-cloneV", Version: "1.0.0", Clone: true, Original: "develop"},
				"develop-clone-2": {Name: "develop-cloneV", Version: "2.0.0", Clone: true, Original: "develop"},
				"release":         {Name: "release", Version: "2.0.0", Clone: false},
				"production":      {Name: "production", Version: "3.0.0", Clone: false},
			},
			expectedParams: []repository.EnvParams{
				{Name: "develop", Version: "2.0.0", Clone: false},
				{Name: "develop-cloneV", Version: "1.0.0", Clone: true, Original: "develop"},
				{Name: "develop-cloneV", Version: "2.0.0", Clone: true, Original: "develop"},
				{Name: "release", Version: "2.0.0", Clone: false},
				{Name: "production", Version: "3.0.0", Clone: false},
			},
			expectError: false,
			expectedErr: nil,
		},
		{
			name:           "No environments found",
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
					jsonData, err := json.Marshal(m)
					suite.NoError(err)

					_, err = suite.client.Put(ctx, suite.keys.GetEnvKey(envName), string(jsonData))
					suite.NoError(err)
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

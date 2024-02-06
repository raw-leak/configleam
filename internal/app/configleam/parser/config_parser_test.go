package parser_test

import (
	"testing"

	"github.com/raw-leak/configleam/internal/app/configleam/parser"
	"github.com/raw-leak/configleam/internal/app/configleam/types"
	"github.com/stretchr/testify/assert"
)

func TestYamlConfigParser(t *testing.T) {
	testCases := []struct {
		name              string
		description       string
		input             types.ExtractedConfigList
		expectedErr       error
		expectedParsedCfg types.ParsedRepoConfig
	}{
		{
			name:        "Test when a groups has directly assigned values of not string type",
			description: "It should treat the any type of int/uint and bool as global key pointer in string format",
			input: types.ExtractedConfigList{
				// file: groups.yaml
				map[string]interface{}{
					"group:app1": 2,
					"group:app2": true,
					"group:app3": 3.25,
					"group:app4": -3.25,
				},
			},
			expectedErr: nil,
			expectedParsedCfg: types.ParsedRepoConfig{
				AllKeys: []string{"group:app1", "group:app2", "group:app3", "group:app4"},
				Groups: map[string]types.GroupConfig{
					"group:app1": {
						Local:  map[string]interface{}{},
						Global: []string{"2"},
					},
					"group:app2": {
						Local:  map[string]interface{}{},
						Global: []string{"true"},
					},
					"group:app3": {
						Local:  map[string]interface{}{},
						Global: []string{"3.25"},
					},
					"group:app4": {
						Local:  map[string]interface{}{},
						Global: []string{"-3.25"},
					},
				},
				Globals: map[string]interface{}{},
			},
		},
		{
			name:        "Test when we have only global variables",
			description: "It should generate only global key-values",
			input: types.ExtractedConfigList{
				// file: global.yaml
				map[string]interface{}{
					"globalKey1": true,
					"globalKey2": map[string]interface{}{
						"key1": "value1",
						"key2": "value2",
					},
					"globalKey3": []interface{}{
						"list1",
						"list2",
						map[string]interface{}{
							"list3": []interface{}{
								"list3.1",
								"list3.2",
							},
						},
					},
					"globalKey4": []interface{}{
						map[string]interface{}{
							"key1.1": "value1",
							"key1.2": "value2",
						},
						map[string]interface{}{
							"key2.1": "value1",
							"key2.2": "value2",
						},
					},
					"globalKey5": []interface{}{
						map[string]interface{}{
							"key1.1": "value1",
							"key1.2": "value2",
						},
						map[string]interface{}{
							"key2.1": "value1",
							"key2.2": "value2",
						},
					},
					"globalKey6": []interface{}{
						[]interface{}{"item1", "item2"},
						[]interface{}{"item3", "item4"},
					},
				},
			},
			expectedErr: nil,
			expectedParsedCfg: types.ParsedRepoConfig{
				AllKeys: []string{"globalKey1", "globalKey2", "globalKey3", "globalKey4", "globalKey5", "globalKey6"},
				Groups:  map[string]types.GroupConfig{},
				Globals: map[string]interface{}{
					"globalKey1": true,
					"globalKey2": map[string]interface{}{
						"key1": "value1",
						"key2": "value2",
					},
					"globalKey3": []interface{}{
						"list1",
						"list2",
						map[string]interface{}{
							"list3": []interface{}{
								"list3.1",
								"list3.2",
							},
						},
					},
					"globalKey4": []interface{}{
						map[string]interface{}{
							"key1.1": "value1",
							"key1.2": "value2",
						},
						map[string]interface{}{
							"key2.1": "value1",
							"key2.2": "value2",
						},
					},
					"globalKey5": []interface{}{
						map[string]interface{}{
							"key1.1": "value1",
							"key1.2": "value2",
						},
						map[string]interface{}{
							"key2.1": "value1",
							"key2.2": "value2",
						},
					},
					"globalKey6": []interface{}{
						[]interface{}{"item1", "item2"},
						[]interface{}{"item3", "item4"},
					},
				},
			},
		},
		{
			name:        "Test with config equal to the one /test-1 git repository would generate, with two global variables and two groups",
			description: "It should generate the right parsed config for the /test-1 git repository scenario, with two global variables and two groups",

			input: types.ExtractedConfigList{
				// file: global.yaml
				map[string]interface{}{
					"globalKey1": map[string]interface{}{"key1": "value1", "key2": "value2"},
					"globalKey2": map[string]interface{}{"key1": "value1", "key2": "value2"},
				},
				// file: groups.yaml
				map[string]interface{}{
					"group:app1": []interface{}{
						"globalKey1",
						"globalKey2",
						map[string]interface{}{
							"localKey1": map[string]interface{}{"key1": "value1", "key2": 2},
						},
						map[string]interface{}{
							"localKey2": true,
						},
					},
					"group:app2": true,
				},
			},
			expectedErr: nil,
			expectedParsedCfg: types.ParsedRepoConfig{
				AllKeys: []string{"globalKey1", "globalKey2", "group:app1", "group:app2"},
				Groups: map[string]types.GroupConfig{
					"group:app1": {
						Local: map[string]interface{}{
							"localKey1": map[string]interface{}{"key1": "value1", "key2": 2},
							"localKey2": true,
						},
						Global: []string{"globalKey1", "globalKey2"},
					},
					"group:app2": {
						Local:  map[string]interface{}{},
						Global: []string{"true"},
					},
				},
				Globals: map[string]interface{}{
					"globalKey1": map[string]interface{}{"key1": "value1", "key2": "value2"},
					"globalKey2": map[string]interface{}{"key1": "value1", "key2": "value2"},
				},
			},
		},
		{
			name:        "Test with config equal to the one /test-2 git repository would generate, with two global variables and two groups",
			description: "It should generate the right parsed config for the /test-2 git repository scenario, with six global variables and two groups",
			input: types.ExtractedConfigList{
				// file: global.yaml
				map[string]interface{}{
					"globalKey1": true,
					"globalKey2": map[string]interface{}{
						"key1": "value1",
						"key2": "value2",
					},
					"globalKey3": []interface{}{
						"list1",
						"list2",
						map[string]interface{}{
							"list3": []interface{}{
								"list3.1",
								"list3.2",
							},
						},
					},
					"globalKey4": []interface{}{
						map[string]interface{}{
							"key1.1": "value1",
							"key1.2": "value2",
						},
						map[string]interface{}{
							"key2.1": "value1",
							"key2.2": "value2",
						},
					},
					"globalKey5": []interface{}{
						map[string]interface{}{
							"key1.1": "value1",
							"key1.2": "value2",
						},
						map[string]interface{}{
							"key2.1": "value1",
							"key2.2": "value2",
						},
					},
					"globalKey6": []interface{}{
						[]interface{}{"item1", "item2"},
						[]interface{}{"item3", "item4"},
					},
				},
				// file: groups.yaml
				map[string]interface{}{
					"group:app1": []interface{}{
						"globalKey1",
						"globalKey2",
						"globalKey3",
						"globalKey4",
						"globalKey5",
						"globalKey6",
					},
					"group:app2": map[string]interface{}{
						"local1": map[string]interface{}{
							"key1": true,
							"key2": map[string]interface{}{
								"key2.1": "value2.1",
								"key2.2": "value2.2",
							},
						},
						"local2": []interface{}{"item1", "item2", "item3"},
					},
				},
			},
			expectedErr: nil,
			expectedParsedCfg: types.ParsedRepoConfig{
				AllKeys: []string{"globalKey1", "globalKey2", "globalKey3", "globalKey4", "globalKey5", "globalKey6", "group:app1", "group:app2"},
				Groups: map[string]types.GroupConfig{
					"group:app1": {
						Local:  map[string]interface{}{},
						Global: []string{"globalKey1", "globalKey2", "globalKey3", "globalKey4", "globalKey5", "globalKey6"},
					},
					"group:app2": {
						Local: map[string]interface{}{
							"local1": map[string]interface{}{
								"key1": true,
								"key2": map[string]interface{}{
									"key2.1": "value2.1",
									"key2.2": "value2.2",
								},
							},
							"local2": []interface{}{"item1", "item2", "item3"},
						},
						Global: []string{},
					},
				},
				Globals: map[string]interface{}{
					"globalKey1": true,
					"globalKey2": map[string]interface{}{
						"key1": "value1",
						"key2": "value2",
					},
					"globalKey3": []interface{}{
						"list1",
						"list2",
						map[string]interface{}{
							"list3": []interface{}{
								"list3.1",
								"list3.2",
							},
						},
					},
					"globalKey4": []interface{}{
						map[string]interface{}{
							"key1.1": "value1",
							"key1.2": "value2",
						},
						map[string]interface{}{
							"key2.1": "value1",
							"key2.2": "value2",
						},
					},
					"globalKey5": []interface{}{
						map[string]interface{}{
							"key1.1": "value1",
							"key1.2": "value2",
						},
						map[string]interface{}{
							"key2.1": "value1",
							"key2.2": "value2",
						},
					},
					"globalKey6": []interface{}{
						[]interface{}{"item1", "item2"},
						[]interface{}{"item3", "item4"},
					},
				},
			},
		},
		{
			name:        "Test with config equal to the one /test-3 git repository would generate, with two global variables and two groups",
			description: "It should generate the right parsed config for the /test-3 git repository scenario, with four global variables and three groups",
			input: types.ExtractedConfigList{
				// file: global.yaml
				map[string]interface{}{
					"database": map[string]interface{}{
						"primary": map[string]interface{}{
							"host": "global-db-host-primary",
							"port": 3306,
							"credentials": map[string]interface{}{
								"username": "dbuser",
								"password": "dbpass",
							},
						},
						"secondary": map[string]interface{}{
							"host": "global-db-host-secondary",
							"port": 3307,
						},
					},
					"logging": map[string]interface{}{
						"level":  "info",
						"format": "json",
					},
					"services": []interface{}{
						map[string]interface{}{
							"name":    "authService",
							"url":     "http://auth.service",
							"timeout": 30,
						},
						map[string]interface{}{
							"name":    "paymentService",
							"url":     "http://payment.service",
							"timeout": 45,
						},
					},
					"features": map[string]interface{}{
						"featureX": true,
						"featureY": map[string]interface{}{
							"enabled":  true,
							"variants": []interface{}{"A", "B", "C"},
						},
					},
				},
				// file: groups1.yaml
				map[string]interface{}{
					"group:analytics": []interface{}{
						map[string]interface{}{
							"database": map[string]interface{}{
								"primary": map[string]interface{}{
									"port": 3310,
								},
							},
						},
						"logging",
					},
					"group:marketing": []interface{}{
						"database",
						map[string]interface{}{
							"logging": map[string]interface{}{
								"level": "debug",
							},
						},
						map[string]interface{}{
							"services": []interface{}{
								map[string]interface{}{
									"name":    "marketingService",
									"url":     "http://marketing.service",
									"timeout": 60,
								},
							},
						},
						"features",
					},
					"group:sales": []interface{}{
						map[string]interface{}{
							"database": map[string]interface{}{
								"secondary": map[string]interface{}{
									"host": "sales-db-host",
									"credentials": map[string]interface{}{
										"username": "salesuser",
										"password": "salespass",
									},
								},
							},
						},
						map[string]interface{}{
							"logging": map[string]interface{}{
								"format": "text",
							},
						},
						map[string]interface{}{
							"services": []interface{}{
								map[string]interface{}{
									"name":    "salesService",
									"url":     "http://sales.service",
									"timeout": 20,
								},
							},
						},
						map[string]interface{}{
							"features": map[string]interface{}{
								"featureY": map[string]interface{}{
									"variants": []interface{}{"D", "E"},
								},
							},
						},
					},
				},
				// file: groups2.yaml
				map[string]interface{}{
					"group:analytics": []interface{}{
						"services",
						map[string]interface{}{
							"features": map[string]interface{}{
								"featureX": false,
							},
						},
					},
				},
			},
			expectedErr: nil,
			expectedParsedCfg: types.ParsedRepoConfig{
				AllKeys: []string{"database", "features", "group:analytics", "group:marketing", "group:sales", "logging", "services"},
				Groups: map[string]types.GroupConfig{
					"group:analytics": {
						Local: map[string]interface{}{
							"database": map[string]interface{}{
								"primary": map[string]interface{}{
									"port": 3310,
								},
							},
							"features": map[string]interface{}{
								"featureX": false,
							},
						},
						Global: []string{"logging", "services"},
					},
					"group:marketing": {
						Local: map[string]interface{}{
							"logging": map[string]interface{}{
								"level": "debug",
							},
							"services": []interface{}{
								map[string]interface{}{
									"name":    "marketingService",
									"url":     "http://marketing.service",
									"timeout": 60,
								},
							},
						},
						Global: []string{"database", "features"},
					},
					"group:sales": {
						Local: map[string]interface{}{
							"database": map[string]interface{}{
								"secondary": map[string]interface{}{
									"host": "sales-db-host",
									"credentials": map[string]interface{}{
										"username": "salesuser",
										"password": "salespass",
									},
								},
							},
							"logging": map[string]interface{}{
								"format": "text",
							},
							"services": []interface{}{
								map[string]interface{}{
									"name":    "salesService",
									"url":     "http://sales.service",
									"timeout": 20,
								},
							},
							"features": map[string]interface{}{
								"featureY": map[string]interface{}{
									"variants": []interface{}{"D", "E"},
								},
							},
						},
						Global: []string{},
					},
				},
				Globals: map[string]interface{}{
					"database": map[string]interface{}{
						"primary": map[string]interface{}{
							"host": "global-db-host-primary",
							"port": 3306,
							"credentials": map[string]interface{}{
								"username": "dbuser",
								"password": "dbpass",
							},
						},
						"secondary": map[string]interface{}{
							"host": "global-db-host-secondary",
							"port": 3307,
						},
					},
					"logging": map[string]interface{}{
						"level":  "info",
						"format": "json",
					},
					"services": []interface{}{
						map[string]interface{}{
							"name":    "authService",
							"url":     "http://auth.service",
							"timeout": 30,
						},
						map[string]interface{}{
							"name":    "paymentService",
							"url":     "http://payment.service",
							"timeout": 45,
						},
					},
					"features": map[string]interface{}{
						"featureX": true,
						"featureY": map[string]interface{}{
							"enabled":  true,
							"variants": []interface{}{"A", "B", "C"},
						},
					},
				},
			},
		},
	}

	p := parser.New()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsedCfg, err := p.ParseConfigList(&tc.input)

			assert.Equal(t, tc.expectedErr, err, "Error should match")
			assert.Equal(t, tc.expectedParsedCfg, *parsedCfg, tc.description)
		})
	}
}

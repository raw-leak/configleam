package extractor_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/raw-leak/configleam/internal/app/configleam/extractor"
	"github.com/raw-leak/configleam/internal/app/configleam/types"
	"github.com/stretchr/testify/assert"
)

func TestExtractConfigList(t *testing.T) {
	tests := []struct {
		name               string
		description        string
		testDir            string
		expectedErr        error
		expectedConfigList types.ExtractedConfigList
	}{
		{
			name:        "Reading only global.yaml and groups.yaml files from /test-1 folder",
			description: "Should generate expected configuration list for /test-1 folder files",
			testDir:     "test-1",
			expectedErr: nil,
			expectedConfigList: types.ExtractedConfigList{
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
		},
		{
			name:        "Reading only global.yaml and groups.yaml files from /test-2 folder",
			description: "Should generate expected configuration list for /test-2 folder files",
			testDir:     "test-2",
			expectedErr: nil,
			expectedConfigList: types.ExtractedConfigList{
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
		},
		{
			name:        "Reading only global.yaml and groups.yaml files from /test-3 folder",
			description: "Should generate expected configuration list for /test-3 folder files",
			testDir:     "test-3",
			expectedErr: nil,
			expectedConfigList: types.ExtractedConfigList{
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
				// file: groups.yaml
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
						"services",
						map[string]interface{}{
							"features": map[string]interface{}{
								"featureX": false,
							},
						},
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
			},
		},
		{
			name:        "Reading only global.yaml and groups.yml files from /test-4 folder",
			description: "Should generate expected configuration list for /test-4 folder files",
			testDir:     "test-4",
			expectedErr: nil,
			expectedConfigList: types.ExtractedConfigList{
				// file: global.yml
				map[string]interface{}{
					"globalKey1": map[string]interface{}{"key1": "value1", "key2": "value2"},
					"globalKey2": map[string]interface{}{"key1": "value1", "key2": "value2"},
				},
				// file: groups.yml
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
		},
		{
			name:        "Reading only global.yml and groups.yml files from /test-5 folder",
			description: "Should generate expected configuration list for /test-5 folder files",
			testDir:     "test-5",
			expectedErr: nil,
			expectedConfigList: types.ExtractedConfigList{
				// file: global.yml
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
				// file: groups.yml
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
		},
		{
			name:        "Reading only global.yml and groups.yml files from /test-6 folder",
			description: "Should generate expected configuration list for /test-6 folder files",
			testDir:     "test-6",
			expectedErr: nil,
			expectedConfigList: types.ExtractedConfigList{
				// file: global.yml
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
				// file: groups.yml
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
						"services",
						map[string]interface{}{
							"features": map[string]interface{}{
								"featureX": false,
							},
						},
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
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			e := extractor.New()

			cwd, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current working directory: %v", err)
			}

			dir := fmt.Sprintf("%s/testdata/%s", cwd, tc.testDir)

			// Act
			configList, err := e.ExtractConfigList(dir)

			// Assert
			assert.Equal(t, tc.expectedErr, err, "Error should match")
			assert.Equal(t, tc.expectedConfigList, *configList, tc.description)
		})
	}
}

package extractor

import (
	"os"

	"gopkg.in/yaml.v3"
)

func processYAMLFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config, nil
}

var yamlProcessor = fileProcessor{
	Extensions: []string{".yaml", ".yml"},
	Process:    processYAMLFile,
}

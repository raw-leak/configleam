package extractor

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/raw-leak/configleam/internal/app/configuration/types"
)

type fileProcessor struct {
	Extensions []string
	Process    func(string) (map[string]interface{}, error)
}

type configExtractor struct {
	processors []fileProcessor
}

func New() *configExtractor {
	return &configExtractor{
		processors: []fileProcessor{yamlProcessor},
	}
}

func (e *configExtractor) ExtractConfigList(dir string) (*types.ExtractedConfigList, error) {
	var configs types.ExtractedConfigList

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			for _, processor := range e.processors {
				if isFileSupported(info.Name(), processor.Extensions) {
					config, err := processor.Process(path)
					if err != nil {
						return err
					}
					configs = append(configs, config)
					break
				}
			}
		}

		return nil
	})

	return &configs, err
}

func isFileSupported(filename string, extensions []string) bool {
	for _, ext := range extensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}

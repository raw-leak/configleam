package analyzer

import (
	"fmt"
	"log"
	"strings"

	"github.com/raw-leak/configleam/internal/app/configleam/gitmanager"
	"github.com/raw-leak/configleam/internal/app/configleam/helper"
)

type EnvUpdate struct {
	Name   string
	Tag    string
	SemVer helper.SemanticVersion
}

type TagAnalyzer struct{}

func New() *TagAnalyzer {
	return &TagAnalyzer{}
}

func (a *TagAnalyzer) AnalyzeTagsForUpdates(envs map[string]gitmanager.Env, tags []string) ([]EnvUpdate, bool, error) {
	envMap := make(map[string]EnvUpdate)

	for _, tag := range tags {
		log.Println("received this tag", tag)

		for envName, env := range envs {
			if strings.HasSuffix(tag, fmt.Sprintf("-%s", envName)) {
				semVer, err := helper.ExtractSemanticVersionFromTag(tag)
				log.Printf("extracted this sem-ver %v from this tag %s\n", envMap, tag)

				if err != nil {
					log.Printf("error on extracting version from the tag [%s]: %v\n", tag, err)
					continue
				}

				if semVer.IsGreaterThan(env.SemVer) {
					if v, ok := envMap[envName]; !ok || semVer.IsGreaterThan(v.SemVer) {
						envMap[envName] = EnvUpdate{Name: envName, Tag: tag, SemVer: semVer}
					}
				}
			}
		}
	}

	if len(envMap) < 1 {
		return []EnvUpdate{}, false, nil
	}

	envUpdates := make([]EnvUpdate, 0, len(envMap))
	for _, evnUpdated := range envMap {
		envUpdates = append(envUpdates, evnUpdated)
	}

	return envUpdates, true, nil
}

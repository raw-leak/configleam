package parser

import (
	"log"
	"reflect"
	"sort"
	"strings"

	"github.com/emirpasic/gods/utils"
	"github.com/raw-leak/configleam/internal/app/configuration/helper"
	"github.com/raw-leak/configleam/internal/app/configuration/types"
)

type configParser struct{}

func New() *configParser {
	return &configParser{}
}

// returns: allKeys, keyList, custom cfg, err
// TODO: should I return already Marshalled/Stringified values?
// - A single file could contain zero or multiple groups with zero or multiple local key-values and zero or multiple global key-values
func (p *configParser) ParseConfigList(repoConfigList *types.ExtractedConfigList) (*types.ParsedRepoConfig, error) {
	parsedCfg := types.ParsedRepoConfig{Globals: map[string]interface{}{}, Groups: map[string]types.GroupConfig{}, AllKeys: []string{}}

	// Now the question is how should we store that data in etcd to be able to request this data in an efficient way. Because the request will look like this:
	// -> We will get:
	// 1. the name of the group/s (optional) (could be multiple groups):
	// 		- Each group will have assigned local keys with its values and global keys that point to the global values
	// 2. the names of the global keys (optional) (could be multiple vars)
	// 		- It will be global key and we need to return the key with its values
	// The difference between keys and values, keys are string pointing to the global values, and the values actually has the value of the key.
	// Local have keys as well with its own values, but they are only visible/accessible in the context of a group
	// How to look up for group's vars.

	// 1. We need to request all the global keys assigned to the group
	// 		- We could save it like this:
	// 		- TODO:

	for _, config := range *repoConfigList {
		for key, value := range config {
			// we store all the keys we are generating in this file
			if ok := helper.Contains(parsedCfg.AllKeys, key); !ok {
				parsedCfg.AllKeys = append(parsedCfg.AllKeys, key)
			}

			if strings.HasPrefix(key, "group:") {
				groupCfg, ok := parsedCfg.Groups[key]
				if !ok {
					groupCfg = types.GroupConfig{
						Local:  map[string]interface{}{},
						Global: []string{},
					}
				}

				if cfgList, ok := value.([]interface{}); ok {
					for _, cfg := range cfgList {
						if s, ok := cfg.(string); ok {
							// if string, it must be a global key pointer:
							// group:<name>:
							// - globalKey
							groupCfg.Global = append(groupCfg.Global, s)
							continue
						}

						if m, ok := cfg.(map[string]interface{}); ok {
							// if map, it must be a local key-value configuration where value could be anything:
							// group:<name>:
							// - localKey: <any>
							for mkey, mvalue := range m {
								mskey := utils.ToString(mkey)
								groupCfg.Local[mskey] = mvalue
							}
							continue
						}

						log.Printf("while parsing file config, received unhandled group's key '%s' with value of type %v", key, reflect.TypeOf(cfg))
					}

				} else if m, ok := value.(map[string]interface{}); ok {
					// if map, this group has only local key-values:
					// group:<name>:
					// - localKey: <any>
					for mkey, mvalue := range m {
						mskey := utils.ToString(mkey)
						groupCfg.Local[mskey] = mvalue
					}

				} else if s, ok := value.(string); ok {
					// if group has only one global key pointer assigned
					// group:<name>: string
					groupCfg.Global = append(groupCfg.Global, s)

				} else if num, ok := helper.IsAnyNumber(value); ok {
					// if group has only one global key pointer assigned
					// group:<name>: string
					groupCfg.Global = append(groupCfg.Global, utils.ToString(num))

				} else if b, ok := value.(bool); ok {
					// if group has only one global key pointer assigned
					// group:<name>: true
					groupCfg.Global = append(groupCfg.Global, utils.ToString(b))

				} else {
					log.Printf("while parsing file config, received unhandled group's value '%s' with value of type %v", key, reflect.TypeOf(value))
				}

				parsedCfg.Groups[key] = groupCfg
			} else {
				// global configuration:
				// globalKey: <any>
				parsedCfg.Globals[key] = value
			}
		}
	}

	// sort to keep the order always equal
	sort.Strings(parsedCfg.AllKeys)

	return &parsedCfg, nil
}

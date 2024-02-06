package analyzer_test

import (
	"testing"

	"github.com/raw-leak/configleam/internal/app/configleam/analyzer"
	"github.com/raw-leak/configleam/internal/app/configleam/gitmanager"
	"github.com/raw-leak/configleam/internal/app/configleam/helper"
	"github.com/stretchr/testify/assert"
)

func TestAnalyzeTagsForUpdates(t *testing.T) {
	testCases := []struct {
		name               string
		envs               map[string]gitmanager.Env
		tags               []string
		expectedUpdates    []analyzer.EnvUpdate
		expectedHasUpdates bool
		expectedErr        error
	}{
		{
			"No new tags",
			map[string]gitmanager.Env{
				"develop": {LastTag: "v1.0.0-develop", SemVer: helper.SemanticVersion{Major: 1, Minor: 0, Patch: 0}},
			},
			[]string{"v1.0.0-develop"},
			[]analyzer.EnvUpdate{},
			false,
			nil,
		},
		{
			"Single new tag",
			map[string]gitmanager.Env{
				"develop": {LastTag: "v1.0.0-develop", SemVer: helper.SemanticVersion{Major: 1, Minor: 0, Patch: 0}},
			},
			[]string{"v1.1.0-develop"},
			[]analyzer.EnvUpdate{{Name: "develop", Tag: "v1.1.0-develop", SemVer: helper.SemanticVersion{Major: 1, Minor: 1, Patch: 0}}},
			true,
			nil,
		},
		{
			"Multiple environments",
			map[string]gitmanager.Env{
				"develop": {LastTag: "v1.0.0-develop", SemVer: helper.SemanticVersion{Major: 1, Minor: 0, Patch: 0}},
				"staging": {LastTag: "v1.0.0-staging", SemVer: helper.SemanticVersion{Major: 1, Minor: 0, Patch: 0}},
			},
			[]string{"v1.1.0-develop", "v1.2.0-staging"},
			[]analyzer.EnvUpdate{
				{Name: "develop", Tag: "v1.1.0-develop", SemVer: helper.SemanticVersion{Major: 1, Minor: 1, Patch: 0}},
				{Name: "staging", Tag: "v1.2.0-staging", SemVer: helper.SemanticVersion{Major: 1, Minor: 2, Patch: 0}},
			},
			true,
			nil,
		},
		{
			"Invalid tag format",
			map[string]gitmanager.Env{
				"develop": {LastTag: "v1.0.0-develop", SemVer: helper.SemanticVersion{Major: 1, Minor: 0, Patch: 0}},
			},
			[]string{"1.1.0-develop"},
			[]analyzer.EnvUpdate{},
			false,
			nil,
		},
		{
			"New and old tags mixed",
			map[string]gitmanager.Env{
				"develop": {LastTag: "v1.0.0-develop", SemVer: helper.SemanticVersion{Major: 1, Minor: 0, Patch: 0}},
			},
			[]string{"v1.1.0-develop", "v1.0.0-develop"},
			[]analyzer.EnvUpdate{{"develop", "v1.1.0-develop", helper.SemanticVersion{Major: 1, Minor: 1, Patch: 0}}},
			true,
			nil,
		},
		{
			"Multiple tags, single environment, one new",
			map[string]gitmanager.Env{
				"develop": {LastTag: "v1.0.0-develop", SemVer: helper.SemanticVersion{Major: 1, Minor: 0, Patch: 0}},
			},
			[]string{"v1.1.0-develop", "v1.0.1-develop"},
			[]analyzer.EnvUpdate{{"develop", "v1.1.0-develop", helper.SemanticVersion{Major: 1, Minor: 1, Patch: 0}}},
			true,
			nil,
		},
		{
			"New tag for unstaged environment",
			map[string]gitmanager.Env{
				"develop": {LastTag: "v1.0.0-develop", SemVer: helper.SemanticVersion{Major: 1, Minor: 0, Patch: 0}},
			},
			[]string{"v1.1.0-feature"},
			[]analyzer.EnvUpdate{},
			false,
			nil,
		},
		{
			"No environments provided",
			map[string]gitmanager.Env{},
			[]string{"v1.1.0-develop"},
			[]analyzer.EnvUpdate{},
			false,
			nil,
		},
		{
			"Environment with no last tag",
			map[string]gitmanager.Env{
				"develop": {LastTag: "", SemVer: helper.SemanticVersion{Major: 0, Minor: 0, Patch: 0}},
			},
			[]string{"v1.0.0-develop"},
			[]analyzer.EnvUpdate{{"develop", "v1.0.0-develop", helper.SemanticVersion{Major: 1, Minor: 0, Patch: 0}}},
			true,
			nil,
		},
		{
			"Tag for an environment with a higher patch version",
			map[string]gitmanager.Env{
				"develop": {LastTag: "v1.0.0-develop", SemVer: helper.SemanticVersion{Major: 1, Minor: 0, Patch: 0}},
			},
			[]string{"v1.0.1-develop"},
			[]analyzer.EnvUpdate{{"develop", "v1.0.1-develop", helper.SemanticVersion{Major: 1, Minor: 0, Patch: 1}}},
			true,
			nil,
		},
		{
			"Multiple environments, some with no updates",
			map[string]gitmanager.Env{
				"develop": {LastTag: "v1.0.0-develop", SemVer: helper.SemanticVersion{Major: 1, Minor: 0, Patch: 0}},
				"staging": {LastTag: "v1.0.0-staging", SemVer: helper.SemanticVersion{Major: 1, Minor: 0, Patch: 0}},
			},
			[]string{"v1.1.0-develop"},
			[]analyzer.EnvUpdate{{"develop", "v1.1.0-develop", helper.SemanticVersion{Major: 1, Minor: 1, Patch: 0}}},
			true,
			nil,
		},
		{
			"Tag with higher major version",
			map[string]gitmanager.Env{
				"develop": {LastTag: "v1.0.0-develop", SemVer: helper.SemanticVersion{Major: 1, Minor: 0, Patch: 0}},
			},
			[]string{"v2.0.0-develop"},
			[]analyzer.EnvUpdate{{"develop", "v2.0.0-develop", helper.SemanticVersion{Major: 2, Minor: 0, Patch: 0}}},
			true,
			nil,
		},
		{
			"Tag with non-numeric version components",
			map[string]gitmanager.Env{
				"develop": {LastTag: "v1.0.0-develop", SemVer: helper.SemanticVersion{Major: 1, Minor: 0, Patch: 0}},
			},
			[]string{"v1.x.y-develop"},
			[]analyzer.EnvUpdate{},
			false,
			nil,
		},
		{
			"Empty tag list",
			map[string]gitmanager.Env{
				"develop": {LastTag: "v1.0.0-develop", SemVer: helper.SemanticVersion{Major: 1, Minor: 0, Patch: 0}},
			},
			[]string{},
			[]analyzer.EnvUpdate{},
			false,
			nil,
		},
		{
			"Identify Newest Tags from a Set of Old and New Tags",
			map[string]gitmanager.Env{
				"develop":    {LastTag: "v1.0.5-develop", SemVer: helper.SemanticVersion{Major: 1, Minor: 0, Patch: 5}},
				"staging":    {LastTag: "v2.1.5-staging", SemVer: helper.SemanticVersion{Major: 2, Minor: 1, Patch: 5}},
				"production": {LastTag: "v3.2.5-production", SemVer: helper.SemanticVersion{Major: 3, Minor: 2, Patch: 5}},
			},
			[]string{
				// Old tags
				"v1.0.0-develop", "v1.0.1-develop", "v1.0.2-develop", "v1.0.3-develop", "v1.0.4-develop",
				"v2.1.0-staging", "v2.1.1-staging", "v2.1.2-staging", "v2.1.3-staging", "v2.1.4-staging",
				"v3.2.0-production", "v3.2.1-production", "v3.2.2-production", "v3.2.3-production", "v3.2.4-production",
				// Newer tags
				"v1.0.6-develop", "v1.0.7-develop", "v1.0.8-develop", "v1.0.9-develop", "v1.1.0-develop",
				"v2.1.6-staging", "v2.1.7-staging", "v2.1.8-staging", "v2.1.9-staging", "v2.2.0-staging",
				"v3.2.6-production", "v3.2.7-production", "v3.2.8-production", "v3.2.9-production", "v3.3.0-production",
			},
			[]analyzer.EnvUpdate{
				{"develop", "v1.1.0-develop", helper.SemanticVersion{Major: 1, Minor: 1, Patch: 0}},
				{"staging", "v2.2.0-staging", helper.SemanticVersion{Major: 2, Minor: 2, Patch: 0}},
				{"production", "v3.3.0-production", helper.SemanticVersion{Major: 3, Minor: 3, Patch: 0}},
			},
			true,
			nil,
		},
		{
			"Include Invalid Tags and Ensure They Are Skipped",

			map[string]gitmanager.Env{
				"develop": {LastTag: "v1.0.5-develop", SemVer: helper.SemanticVersion{Major: 1, Minor: 0, Patch: 5}},
				"staging": {LastTag: "v2.1.5-staging", SemVer: helper.SemanticVersion{Major: 2, Minor: 1, Patch: 5}},
			},
			[]string{
				// Invalid tags
				"v1.x-develop", "v2.y-staging",
				// Old valid tags
				"v1.0.4-develop", "v2.1.4-staging",
				// Valid newer tags
				"v1.0.6-develop", "v2.1.6-staging",
			},
			[]analyzer.EnvUpdate{
				{"develop", "v1.0.6-develop", helper.SemanticVersion{Major: 1, Minor: 0, Patch: 6}},
				{"staging", "v2.1.6-staging", helper.SemanticVersion{Major: 2, Minor: 1, Patch: 6}},
			},
			true,
			nil,
		},
	}

	anlzr := analyzer.New()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			updates, hasUpdates, err := anlzr.AnalyzeTagsForUpdates(tc.envs, tc.tags)

			assert.Equal(t, tc.expectedErr, err, "Error should match")
			assert.Equal(t, tc.expectedHasUpdates, hasUpdates, "Expected has updates to match (Ok-idiom )")
			assert.Equal(t, tc.expectedUpdates, updates, "Expected updates to match")

		})
	}
}

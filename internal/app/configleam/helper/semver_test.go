package helper_test

import (
	"testing"

	"github.com/raw-leak/configleam/internal/app/configleam/helper"
)

func TestExtractSemanticVersionFromTag(t *testing.T) {
	testCases := []struct {
		tag           string
		expectedVer   helper.SemanticVersion
		expectedError bool
	}{
		{"v1.2.3", helper.SemanticVersion{Major: 1, Minor: 2, Patch: 3}, false},
		{"v0.10.0", helper.SemanticVersion{Major: 0, Minor: 10, Patch: 0}, false},
		{"v123.456.789", helper.SemanticVersion{Major: 123, Minor: 456, Patch: 789}, false},
		{"v1.0", helper.SemanticVersion{}, true},
		{"1.2.3", helper.SemanticVersion{}, true},
	}

	for _, tc := range testCases {
		ver, err := helper.ExtractSemanticVersionFromTag(tc.tag)
		if tc.expectedError {
			if err == nil {
				t.Errorf("Expected an error for tag '%s'", tc.tag)
			}
		} else {
			if err != nil {
				t.Errorf("Did not expect an error for tag '%s': %v", tc.tag, err)
			}
			if ver != tc.expectedVer {
				t.Errorf("Expected version %+v for tag '%s', got %+v", tc.expectedVer, tc.tag, ver)
			}
		}
	}
}

func TestIsGreaterThan(t *testing.T) {
	testCases := []struct {
		v1              helper.SemanticVersion
		v2              helper.SemanticVersion
		expectedGreater bool
	}{
		{helper.SemanticVersion{Major: 1, Minor: 2, Patch: 3}, helper.SemanticVersion{Major: 1, Minor: 2, Patch: 2}, true},
		{helper.SemanticVersion{Major: 1, Minor: 1, Patch: 1}, helper.SemanticVersion{Major: 1, Minor: 1, Patch: 1}, false},
		{helper.SemanticVersion{Major: 1, Minor: 0, Patch: 0}, helper.SemanticVersion{Major: 1, Minor: 1, Patch: 0}, false},
		{helper.SemanticVersion{Major: 2, Minor: 0, Patch: 0}, helper.SemanticVersion{Major: 1, Minor: 9, Patch: 9}, true},
		{helper.SemanticVersion{Major: 1, Minor: 2, Patch: 3}, helper.SemanticVersion{Major: 1, Minor: 2, Patch: 3}, false},
		{helper.SemanticVersion{Major: 0, Minor: 9, Patch: 9}, helper.SemanticVersion{Major: 1, Minor: 0, Patch: 0}, false},
	}

	for _, tc := range testCases {
		if greater := tc.v1.IsGreaterThan(tc.v2); greater != tc.expectedGreater {
			t.Errorf("Expected %v.IsGreaterThan(%v) to be %v, got %v", tc.v1, tc.v2, tc.expectedGreater, greater)
		}
	}
}

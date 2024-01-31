package helper

import (
	"fmt"
	"regexp"
	"strconv"
)

type SemanticVersion struct {
	Major int64
	Minor int64
	Patch int64
}

func ExtractSemanticVersionFromTag(tag string) (SemanticVersion, error) {
	var semVer SemanticVersion
	re := regexp.MustCompile(`v(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(tag)

	if len(matches) != 4 {
		return semVer, fmt.Errorf("invalid semantic version format: %s", tag)
	}

	var err error
	if semVer.Major, err = strconv.ParseInt(matches[1], 10, 64); err != nil {
		return semVer, fmt.Errorf("error parsing major version: %w", err)
	}
	if semVer.Minor, err = strconv.ParseInt(matches[2], 10, 64); err != nil {
		return semVer, fmt.Errorf("error parsing minor version: %w", err)
	}
	if semVer.Patch, err = strconv.ParseInt(matches[3], 10, 64); err != nil {
		return semVer, fmt.Errorf("error parsing patch version: %w", err)
	}

	return semVer, nil
}

func (sm SemanticVersion) IsGreaterThan(other SemanticVersion) bool {
	if sm.Major != other.Major {
		return sm.Major > other.Major
	}
	if sm.Minor != other.Minor {
		return sm.Minor > other.Minor
	}
	return sm.Patch > other.Patch
}

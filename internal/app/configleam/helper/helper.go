package helper

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// IsAnyNumber checks if the interface{} is any numeric type.
// Returns the numeric value as float64 (if it is a number) and a bool indicating whether it is a number.
func IsAnyNumber(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

// ExtractVersionFromTag parses a semantic versioning tag string and converts it
// to a float64 value for comparison purposes. The tag is expected to follow the
// format 'v<major>.<minor>.<patch>' (e.g., 'v1.2.3'). It returns an error for
// invalid formats or parsing issues.
func ExtractVersionFromTag(tag string) (float64, error) {
	re := regexp.MustCompile(`v([0-9]+\.[0-9]+\.[0-9]+)`)
	matches := re.FindStringSubmatch(tag)
	if len(matches) != 2 {
		return 0, fmt.Errorf("invalid tag format: %s", tag)
	}
	ver, err := strconv.ParseFloat(strings.ReplaceAll(matches[1], ".", ""), 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing version: %w", err)
	}
	return ver, nil
}

func ExtractRepoNameFromRepoURL(repoURL string) (string, error) {
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("error parsing URL %q: %w", repoURL, err)
	}

	parts := strings.Split(parsedURL.Path, "/")
	if len(parts) > 0 {
		return strings.TrimSuffix(parts[len(parts)-1], ".git"), nil
	}

	return "", fmt.Errorf("could not extract repository name from URL %q", repoURL)
}

func Contains(arr []string, target string) bool {
	for _, v := range arr {
		if v == target {
			return true
		}
	}
	return false
}

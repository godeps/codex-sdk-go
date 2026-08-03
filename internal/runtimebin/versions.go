package runtimebin

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	normalizedVersionPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)*(?:(?:a|b|rc)[0-9]+)?(?:\.post[0-9]+)?$`)
	alphaHotfixPattern       = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)*)a([0-9]+)\.post([0-9]+)$`)
	prereleasePattern        = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)*)(a|b|rc)([0-9]+)$`)
)

func NormalizeRuntimeVersion(version string) (string, error) {
	normalized := strings.TrimSpace(version)
	switch {
	case strings.HasPrefix(normalized, "rust-v"):
		normalized = strings.TrimPrefix(normalized, "rust-v")
	case strings.HasPrefix(normalized, "v"):
		normalized = strings.TrimPrefix(normalized, "v")
	}

	replacements := []struct {
		old *regexp.Regexp
		new string
	}{
		{regexp.MustCompile(`-alpha\.?([0-9]+)\.([0-9]+)$`), `a$1.post$2`},
		{regexp.MustCompile(`-alpha\.?([0-9]+)$`), `a$1`},
		{regexp.MustCompile(`-beta\.?([0-9]+)$`), `b$1`},
		{regexp.MustCompile(`-rc\.?([0-9]+)$`), `rc$1`},
	}
	for _, replacement := range replacements {
		normalized = replacement.old.ReplaceAllString(normalized, replacement.new)
	}

	if !normalizedVersionPattern.MatchString(normalized) {
		return "", fmt.Errorf("runtimebin: unsupported runtime version %q", version)
	}
	return normalized, nil
}

func ReleaseTagForVersion(version string) (string, error) {
	releaseVersion, err := ReleaseVersionForVersion(version)
	if err != nil {
		return "", err
	}
	return "rust-v" + releaseVersion, nil
}

func ReleaseVersionForVersion(version string) (string, error) {
	normalized, err := NormalizeRuntimeVersion(version)
	if err != nil {
		return "", err
	}
	if matches := alphaHotfixPattern.FindStringSubmatch(normalized); matches != nil {
		return fmt.Sprintf("%s-alpha.%s.%s", matches[1], matches[2], matches[3]), nil
	}
	if matches := prereleasePattern.FindStringSubmatch(normalized); matches != nil {
		name := map[string]string{"a": "alpha", "b": "beta", "rc": "rc"}[matches[2]]
		return fmt.Sprintf("%s-%s.%s", matches[1], name, matches[2+1]), nil
	}
	return normalized, nil
}

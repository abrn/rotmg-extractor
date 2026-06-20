// Package extract holds asset-extraction logic that runs against a local build.
package extract

import (
	"os"
	"regexp"
)

// legacyVersionPattern matches the Exalt version the way the original Python
// did: a "127.0.0.1" anchor followed by a five-part version (e.g. 1.3.2.0.0).
// Newer (Unity 6) builds no longer embed the version this way, so this is
// best-effort and may return "".
var legacyVersionPattern = regexp.MustCompile(`127\.0\.0\.1[\x00-\x20]*(\d(?:\.\d){4})`)

// ExaltVersion attempts to read the Exalt version string out of
// global-metadata.dat. It returns ("", nil) when no version can be found, which
// is not treated as an error by callers.
func ExaltVersion(metadataPath string) (string, error) {
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return "", err
	}

	if m := legacyVersionPattern.FindSubmatch(data); m != nil {
		return string(m[1]), nil
	}
	return "", nil
}

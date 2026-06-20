// Package extract holds asset-extraction logic that runs against a local build.
package extract

import (
	"bytes"
	"os"
	"regexp"
)

// versionAnchor precedes the Exalt version in global-metadata.dat. The version
// is stored as a const string in the same static class as some "127.0.0.1"
// const strings, so the anchor reliably locates it.
var versionAnchor = []byte("127.0.0.1")

// versionPattern matches a five-part version with 1-3 digit components
// (e.g. 6.11.0.1.0). The original tool used single-digit components, which no
// longer matches builds like 6.11.x (two-digit minor).
var versionPattern = regexp.MustCompile(`[0-9]{1,3}(?:\.[0-9]{1,3}){4}`)

// anchorWindow is how many bytes after an anchor to search for the version.
const anchorWindow = 24

// ScanVersion scans each file in turn for the Exalt build version and returns
// the first one found. Missing files are skipped. Returns ("", nil) when no
// version is found.
func ScanVersion(paths ...string) (string, error) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		if v := scanAnchored(data); v != "" {
			return v, nil
		}
	}
	return "", nil
}

// scanAnchored finds a version immediately following a "127.0.0.1" anchor.
func scanAnchored(data []byte) string {
	from := 0
	for {
		idx := bytes.Index(data[from:], versionAnchor)
		if idx < 0 {
			return ""
		}
		start := from + idx + len(versionAnchor)
		end := start + anchorWindow
		if end > len(data) {
			end = len(data)
		}
		if m := versionPattern.Find(data[start:end]); m != nil {
			return string(m)
		}
		from = start
	}
}

// ExaltVersion scans a single file (kept for callers that only have the
// metadata path).
func ExaltVersion(metadataPath string) (string, error) {
	return ScanVersion(metadataPath)
}

package unityassets

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"rotmg-extractor/internal/logx"
)

// assetFilePattern matches the Unity SerializedFiles worth scanning, matching
// the original extractor's selection.
var assetFilePattern = regexp.MustCompile(`^(globalgamemanagers(\.assets)?|level\d+|resources\.assets|sharedassets\d+\.assets)$`)

// Extractor extracts Unity TextAssets in pure Go (no external binary).
type Extractor struct {
	Log *logx.Logger
}

// Name identifies the backend.
func (e *Extractor) Name() string { return "native" }

// Available reports whether the extractor can run. The native parser has no
// external dependencies, so it is always available.
func (e *Extractor) Available() bool { return true }

// Extract scans the Unity data directory for SerializedFiles and writes every
// TextAsset under outDir/TextAsset/<name>.<ext>, choosing the extension from the
// content. Duplicate names are disambiguated with a numeric suffix.
func (e *Extractor) Extract(ctx context.Context, dataDir, outDir string) error {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return fmt.Errorf("reading data dir: %w", err)
	}

	textDir := filepath.Join(outDir, "TextAsset")
	if err := os.MkdirAll(textDir, 0o755); err != nil {
		return err
	}

	used := map[string]int{}
	total := 0

	for _, entry := range entries {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if entry.IsDir() || !assetFilePattern.MatchString(entry.Name()) {
			continue
		}
		path := filepath.Join(dataDir, entry.Name())

		sf, err := OpenSerializedFile(path)
		if err != nil {
			e.Log.Debug("skipping %s: %v", entry.Name(), err)
			continue
		}

		assets, err := sf.TextAssets()
		if err != nil {
			e.Log.Warn("failed reading TextAssets from %s: %v", entry.Name(), err)
			continue
		}
		if len(assets) == 0 {
			continue
		}

		for _, ta := range assets {
			name := sanitize(ta.Name)
			if name == "" {
				name = fmt.Sprintf("untitled_%d", used["untitled"])
				used["untitled"]++
			}
			ext := DetectExtension(ta.Script)
			outPath := uniquePath(textDir, name, ext, used)
			if err := os.WriteFile(outPath, ta.Script, 0o644); err != nil {
				return fmt.Errorf("writing %s: %w", outPath, err)
			}
			total++
		}
		e.Log.Info("Extracted %d TextAsset(s) from %s", len(assets), entry.Name())
	}

	e.Log.Info("Native extraction complete: %d TextAsset(s) -> %s", total, textDir)
	return nil
}

// uniquePath builds outDir/name.ext, appending -N if that name+ext is taken.
func uniquePath(dir, name, ext string, used map[string]int) string {
	key := name + "." + ext
	n := used[key]
	used[key] = n + 1
	if n == 0 {
		return filepath.Join(dir, name+"."+ext)
	}
	return filepath.Join(dir, fmt.Sprintf("%s-%d.%s", name, n, ext))
}

// sanitize removes path separators and characters invalid in file names.
func sanitize(name string) string {
	name = strings.TrimSpace(name)
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "", "*", "", "?", "", "\"", "", "<", "", ">", "", "|", "",
	)
	return replacer.Replace(name)
}

// Package fsutil provides small filesystem helpers used across the extractor:
// recursive copying and content hashing.
package fsutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyFile copies src to dst, creating parent directories as needed and
// preserving the source file mode.
func CopyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// CopyDir recursively copies the directory tree rooted at src into dst.
// Symlinks are skipped (RotMG build data contains none of interest).
func CopyDir(src, dst string) error {
	return CopyDirExcept(src, dst, nil)
}

// CopyDirExcept is CopyDir but skips any entry whose path relative to src (slash
// separated) is in the exclude set. A top-level excluded directory is skipped
// entirely.
func CopyDirExcept(src, dst string, exclude map[string]bool) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if exclude[filepath.ToSlash(rel)] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)

		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case d.Type()&os.ModeSymlink != 0:
			return nil // skip symlinks
		default:
			return CopyFile(path, target)
		}
	})
}

// HashFile returns the first 12 hex characters of the file's SHA-256 digest,
// suitable as a short build identity.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil))[:12], nil
}

// Exists reports whether path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// MustRel is filepath.Rel but returns the original target on error, for logging.
func MustRel(base, target string) string {
	if rel, err := filepath.Rel(base, target); err == nil {
		return rel
	}
	return fmt.Sprintf("%s", target)
}

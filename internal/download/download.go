// Package download fetches RotMG build files from the CDN.
//
// Client builds publish a checksum.json manifest listing every file; each file
// is served gzip-compressed (the plain path 4xxs, the ".gz" path serves it), so
// downloads try the plain URL first and fall back to "<path>.gz", decompressing
// on the fly. Downloaded files are verified against the manifest's MD5.
//
// Launcher builds have no manifest: they are a single "<buildURL>.exe" installer.
package download

import (
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"rotmg-extractor/internal/logx"
)

// Manifest is the checksum.json published with each client build.
type Manifest struct {
	Files []File `json:"files"`
}

// File is one entry in the manifest.
type File struct {
	Path     string `json:"file"`
	Checksum string `json:"checksum"` // MD5 of the decompressed file
	Size     int64  `json:"size"`
}

// maxAttempts is how many times a network operation is tried before failing.
const maxAttempts = 4

// withRetry runs fn, retrying transient failures with linear backoff. It stops
// early if the context is cancelled.
func withRetry(ctx context.Context, fn func() error) error {
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}
	}
	return err
}

// FetchManifest retrieves and parses <buildURL>/checksum.json.
func FetchManifest(ctx context.Context, buildURL string) (*Manifest, error) {
	u := strings.TrimRight(buildURL, "/") + "/checksum.json"
	client := &http.Client{Timeout: 30 * time.Second}
	var m Manifest
	err := withRetry(ctx, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("fetching manifest: unexpected status %s", resp.Status)
		}
		var parsed Manifest
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return fmt.Errorf("parsing manifest: %w", err)
		}
		m = parsed
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// FileFilter reports whether a manifest file (by its path) should be
// downloaded. A nil filter means "all files".
type FileFilter func(path string) bool

// Options controls a client download.
type Options struct {
	// Filter selects which files to download; nil downloads everything.
	Filter FileFilter
	// Incremental skips files already present in destDir with a matching
	// checksum, and prunes files no longer in the (filtered) manifest.
	Incremental bool
}

// Stats summarises a download.
type Stats struct {
	Total      int // files selected
	Downloaded int
	Reused     int // skipped because unchanged (incremental)
	Pruned     int
}

// ClientFiles downloads the manifest's files into destDir per the options.
func ClientFiles(ctx context.Context, log *logx.Logger, buildURL, destDir string, opts Options) (Stats, error) {
	m, err := FetchManifest(ctx, buildURL)
	if err != nil {
		return Stats{}, err
	}

	wanted := m.Files
	if opts.Filter != nil {
		wanted = wanted[:0:0]
		for _, f := range m.Files {
			if opts.Filter(f.Path) {
				wanted = append(wanted, f)
			}
		}
	}

	log.Info("Downloading %d of %d build files...", len(wanted), len(m.Files))
	log.Indent()
	defer log.Dedent()

	stats := Stats{Total: len(wanted)}
	keep := make(map[string]bool, len(wanted))
	client := &http.Client{Timeout: 10 * time.Minute}

	for i, f := range wanted {
		if ctx.Err() != nil {
			return stats, ctx.Err()
		}
		dest, err := safeJoin(destDir, f.Path)
		if err != nil {
			return stats, err
		}
		keep[dest] = true

		if opts.Incremental && fileMatches(dest, f.Checksum) {
			stats.Reused++
			log.Debug("(%d/%d) unchanged %s", i+1, len(wanted), f.Path)
			continue
		}
		if err := downloadFile(ctx, client, joinURL(buildURL, f.Path), dest, f.Checksum); err != nil {
			return stats, fmt.Errorf("downloading %s: %w", f.Path, err)
		}
		stats.Downloaded++
		log.Debug("(%d/%d) %s", i+1, len(wanted), f.Path)
	}

	if opts.Incremental {
		stats.Pruned = pruneExtra(destDir, keep)
	}

	log.Info("Downloaded %d, reused %d, pruned %d (of %d selected)",
		stats.Downloaded, stats.Reused, stats.Pruned, stats.Total)
	return stats, nil
}

// fileMatches reports whether the file at path exists with the given MD5.
func fileMatches(path, wantMD5 string) bool {
	if wantMD5 == "" {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}
	return hex.EncodeToString(h.Sum(nil)) == wantMD5
}

// pruneExtra removes files under destDir that are not in keep, and returns the
// number removed. Empty directories are left in place.
func pruneExtra(destDir string, keep map[string]bool) int {
	removed := 0
	filepath.WalkDir(destDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !keep[path] {
			if os.Remove(path) == nil {
				removed++
			}
		}
		return nil
	})
	return removed
}

// installerExts are the launcher installer formats to try, in order. Windows
// ships a .exe, macOS a .dmg; the CDN serves only the right one (others 4xx).
var installerExts = []string{".exe", ".dmg", ".pkg", ".zip"}

// LauncherInstaller downloads the launcher installer into destDir, trying the
// known installer extensions (and a .gz fallback for each), and returns the
// path to the downloaded file.
func LauncherInstaller(ctx context.Context, log *logx.Logger, buildURL, buildID, destDir string) (string, error) {
	log.Info("Downloading launcher installer...")
	client := &http.Client{Timeout: 15 * time.Minute}

	var result string
	err := withRetry(ctx, func() error {
		for _, ext := range installerExts {
			dest := filepath.Join(destDir, buildID+ext)
			ok, err := fetchTo(ctx, client, buildURL+ext, false, dest, "")
			if err != nil {
				return err
			}
			if !ok {
				ok, err = fetchTo(ctx, client, buildURL+ext+".gz", true, dest, "")
				if err != nil {
					return err
				}
			}
			if ok {
				result = dest
				return nil
			}
		}
		return fmt.Errorf("launcher installer not found (tried %v)", installerExts)
	})
	if err != nil {
		return "", err
	}
	log.Info("Downloaded launcher installer (%s)", filepath.Base(result))
	return result, nil
}

// downloadFile fetches baseURL to dest, falling back to baseURL+".gz" (gzip)
// when the plain path is unavailable. When wantMD5 is set the decompressed bytes
// are verified against it. Transient failures (network errors, 5xx, a corrupt
// transfer that fails the checksum) are retried.
func downloadFile(ctx context.Context, client *http.Client, baseURL, dest, wantMD5 string) error {
	return withRetry(ctx, func() error {
		ok, err := fetchTo(ctx, client, baseURL, false, dest, wantMD5)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		// Plain path missing: the file is gzip-compressed at the CDN.
		ok, err = fetchTo(ctx, client, baseURL+".gz", true, dest, wantMD5)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("not found (tried plain and .gz): %s", baseURL)
		}
		return nil
	})
}

// fetchTo GETs url and, on 200, streams it to dest (gunzipping if gz). It
// returns ok=false (no error) on a non-200 status so the caller can try a
// fallback URL.
func fetchTo(ctx context.Context, client *http.Client, url string, gz bool, dest, wantMD5 string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return false, nil
	}

	var src io.Reader = resp.Body
	if gz {
		zr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return true, fmt.Errorf("gunzip %s: %w", url, err)
		}
		defer zr.Close()
		src = zr
	}
	if err := writeVerified(dest, src, wantMD5); err != nil {
		return true, err
	}
	return true, nil
}

// writeVerified streams src to a temp file (computing its MD5), verifies the
// checksum, then atomically renames it into place — so an interrupted transfer
// or a checksum mismatch never leaves a corrupt file at dest.
func writeVerified(dest string, src io.Reader, wantMD5 string) error {
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".download-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once renamed; cleans up on any failure

	h := md5.New()
	if _, err := io.Copy(io.MultiWriter(tmp, h), src); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if wantMD5 != "" {
		if got := hex.EncodeToString(h.Sum(nil)); got != wantMD5 {
			return fmt.Errorf("checksum mismatch for %s: got %s want %s", dest, got, wantMD5)
		}
	}
	return os.Rename(tmpName, dest)
}

// safeJoin resolves a manifest-relative path under base, rejecting paths that
// would escape base (e.g. via "..") — defense against a malicious manifest.
func safeJoin(base, relpath string) (string, error) {
	dest := filepath.Join(base, filepath.FromSlash(relpath))
	rel, err := filepath.Rel(base, dest)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("manifest path %q escapes the destination", relpath)
	}
	return dest, nil
}

// joinURL appends a manifest path to the build URL, escaping path segments
// (e.g. spaces -> %20) while preserving the slashes.
func joinURL(base, relpath string) string {
	u := url.URL{Path: relpath}
	return strings.TrimRight(base, "/") + "/" + u.EscapedPath()
}

// Package assetripper drives a bundled AssetRipper binary to extract Unity
// assets. The distributed AssetRipper build is a self-contained web server
// rather than a one-shot CLI, so this package launches it headless on a local
// port and drives it through its HTTP API:
//
//	POST /LoadFolder              load the build's Data directory
//	POST /Export/PrimaryContent   export decoded assets to an output directory
//
// The server is started and stopped per export.
package assetripper

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"rotmg-extractor/internal/logx"
)

// ResolveBinary returns the path to the AssetRipper executable inside dir,
// using the OS-specific file name.
func ResolveBinary(dir string) string {
	name := "AssetRipper.GUI.Free"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(dir, name)
}

// ExportMode selects which AssetRipper export endpoint to use.
type ExportMode string

const (
	// ExportPrimary exports decoded primary content (assets only). This is the
	// fast path used for pulling out game data and textures.
	ExportPrimary ExportMode = "primary"
	// ExportProject exports a full reconstructed Unity project.
	ExportProject ExportMode = "project"
)

// Client drives a bundled AssetRipper binary.
type Client struct {
	// BinPath is the path to the AssetRipper.GUI.Free executable.
	BinPath string
	// Port is the local port AssetRipper hosts on.
	Port int
	// Mode selects the export endpoint.
	Mode ExportMode
	// Log receives progress messages.
	Log *logx.Logger
}

// Name identifies the backend.
func (c *Client) Name() string { return "assetripper" }

// Extract satisfies the pipeline's extractor interface, exporting with the
// client's configured mode.
func (c *Client) Extract(ctx context.Context, dataDir, outDir string) error {
	return c.Export(ctx, dataDir, outDir, c.Mode)
}

// Available reports whether the AssetRipper binary exists and is runnable.
func (c *Client) Available() bool {
	if c.BinPath == "" {
		return false
	}
	info, err := os.Stat(c.BinPath)
	return err == nil && !info.IsDir()
}

// Export loads inputDir into AssetRipper and exports it to outputDir. outputDir
// is cleared by AssetRipper if it already exists.
func (c *Client) Export(ctx context.Context, inputDir, outputDir string, mode ExportMode) error {
	if !c.Available() {
		return fmt.Errorf("AssetRipper binary not found at %q", c.BinPath)
	}

	// Pick a free port when none is configured, avoiding collisions with a
	// fixed port across runs/instances.
	port := c.Port
	if port == 0 {
		p, err := freePort()
		if err != nil {
			return fmt.Errorf("finding free port: %w", err)
		}
		port = p
	}

	base := "http://127.0.0.1:" + strconv.Itoa(port)
	logPath := filepath.Join(filepath.Dir(outputDir), "assetripper.log")

	// Launch headless. cmd.Dir is set to the binary's directory so its sidecar
	// libraries (e.g. libcapstone.dylib) resolve.
	cmd := exec.CommandContext(ctx, c.BinPath,
		"--headless",
		"--port", strconv.Itoa(port),
		"--log-path", logPath,
	)
	cmd.Dir = filepath.Dir(c.BinPath)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting AssetRipper: %w", err)
	}
	// Ensure the server is always stopped.
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	if err := c.waitReady(ctx, base, 60*time.Second); err != nil {
		return err
	}

	c.Log.Info("Loading build into AssetRipper...")
	if err := postForm(ctx, base+"/LoadFolder", inputDir); err != nil {
		return fmt.Errorf("loading folder: %w", err)
	}

	endpoint := "/Export/PrimaryContent"
	if mode == ExportProject {
		endpoint = "/Export/UnityProject"
	}
	c.Log.Info("Exporting assets via AssetRipper (%s)...", mode)
	if err := postForm(ctx, base+endpoint, outputDir); err != nil {
		return fmt.Errorf("exporting: %w", err)
	}

	// AssetRipper returns HTTP 200 with an in-page error banner on failure (e.g.
	// a bad input path), so a 2xx/3xx is not proof of success. Verify the export
	// actually produced files; if not, surface its log.
	if dirIsEmpty(outputDir) {
		return fmt.Errorf("AssetRipper produced no output; log tail:\n%s", tailFile(logPath, 15))
	}

	c.Log.Info("AssetRipper export finished")
	return nil
}

// freePort returns an available TCP port on the loopback interface.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// dirIsEmpty reports whether dir contains no regular files (missing dir counts
// as empty).
func dirIsEmpty(dir string) bool {
	empty := true
	filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			empty = false
			return filepath.SkipAll
		}
		return nil
	})
	return empty
}

// tailFile returns the last n lines of a file (best effort).
func tailFile(path string, n int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "(no log)"
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// waitReady polls the server root until it responds or the timeout elapses.
func (c *Client) waitReady(ctx context.Context, base string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/", nil)
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
	return fmt.Errorf("AssetRipper did not become ready within %s", timeout)
}

// postForm submits a single "path" form field and treats any non-error,
// sub-400 response (including the server's 302 redirects) as success.
func postForm(ctx context.Context, endpoint, path string) error {
	form := url.Values{"path": {path}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// AssetRipper export is synchronous: it returns only once finished, which
	// can take a while for large builds.
	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s returned %s", endpoint, resp.Status)
	}
	return nil
}

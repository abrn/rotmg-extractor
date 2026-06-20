// Package il2cpp drives external IL2CPP dumpers against a prepared RotMG build.
package il2cpp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"rotmg-extractor/internal/fsutil"
	"rotmg-extractor/internal/logx"
)

const defaultExeName = "RotMGExalt"

var defaultFormats = []string{"dll_il_recovery"}

// Input describes the build files needed by an IL2CPP dumper.
type Input struct {
	AppPath      string
	DataDir      string
	GameAssembly string
	Metadata     string
}

// Cpp2IL drives a bundled Cpp2IL executable.
type Cpp2IL struct {
	BinPath        string
	FullDump       bool
	Formats        []string
	Processors     []string
	ExtraArgs      []string
	Verbose        bool
	Timeout        time.Duration
	Log            *logx.Logger
	ContinueOnFail bool
}

// Name identifies the backend.
func (c *Cpp2IL) Name() string { return "cpp2il" }

// Available reports whether the configured Cpp2IL binary exists.
func (c *Cpp2IL) Available() bool {
	if c == nil || c.BinPath == "" {
		return false
	}
	info, err := os.Stat(c.BinPath)
	return err == nil && !info.IsDir()
}

// Dump stages a minimal game directory and runs Cpp2IL into outDir.
func (c *Cpp2IL) Dump(ctx context.Context, in Input, outDir string) error {
	if !c.Available() {
		return fmt.Errorf("Cpp2IL binary not found at %q", c.BinPath)
	}
	if in.GameAssembly == "" {
		return errors.New("GameAssembly binary not found")
	}
	if in.Metadata == "" {
		return errors.New("dumpable global-metadata.dat not found")
	}

	outDir, err := filepath.Abs(outDir)
	if err != nil {
		return fmt.Errorf("resolving il2cpp dump dir: %w", err)
	}

	if err := os.RemoveAll(outDir); err != nil {
		return fmt.Errorf("clearing il2cpp dump dir: %w", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating il2cpp dump dir: %w", err)
	}

	stageDir := filepath.Join(outDir, "_input")
	if err := c.stageInput(in, stageDir); err != nil {
		return fmt.Errorf("staging Cpp2IL input: %w", err)
	}
	defer os.RemoveAll(stageDir)

	logsDir := filepath.Join(outDir, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return err
	}

	formats := c.formats(ctx, logsDir)
	manifest := manifest{
		Tool:     c.Name(),
		Binary:   c.BinPath,
		FullDump: c.FullDump,
		Inputs: inputManifest{
			AppPath:      in.AppPath,
			DataDir:      in.DataDir,
			GameAssembly: in.GameAssembly,
			Metadata:     in.Metadata,
		},
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}
	manifest.Inputs.GameAssemblyHash, _ = fsutil.HashFile(in.GameAssembly)
	manifest.Inputs.MetadataHash, _ = fsutil.HashFile(in.Metadata)

	var runErrs []string
	for _, format := range formats {
		if c.Log != nil {
			c.Log.Info("Running Cpp2IL output format %q...", format)
		}
		result := c.runFormat(ctx, stageDir, outDir, logsDir, format)
		manifest.Formats = append(manifest.Formats, result)
		if result.Error != "" {
			runErrs = append(runErrs, fmt.Sprintf("%s: %s", format, result.Error))
			if !c.ContinueOnFail {
				break
			}
		}
	}

	manifest.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	if err := writeManifest(filepath.Join(outDir, "manifest.json"), manifest); err != nil {
		return err
	}

	if len(runErrs) > 0 {
		return fmt.Errorf("Cpp2IL failed: %s", strings.Join(runErrs, "; "))
	}
	return nil
}

func (c *Cpp2IL) formats(ctx context.Context, logsDir string) []string {
	configured := cleanFormats(c.Formats)
	if !c.FullDump {
		if len(configured) > 0 {
			return configured
		}
		return slices.Clone(defaultFormats)
	}

	listed, err := c.listOutputFormats(ctx, filepath.Join(logsDir, "list-output-formats.log"))
	if err == nil && len(listed) > 0 {
		return listed
	}
	if c.Log != nil && err != nil {
		c.Log.Warn("could not list Cpp2IL output formats, using configured/default formats: %v", err)
	}
	if len(configured) > 0 {
		return configured
	}
	return slices.Clone(defaultFormats)
}

func (c *Cpp2IL) listOutputFormats(ctx context.Context, logPath string) ([]string, error) {
	runCtx, cancel := c.withTimeout(ctx)
	defer cancel()

	cmd := exec.CommandContext(runCtx, c.BinPath, "--list-output-formats")
	cmd.Dir = filepath.Dir(c.BinPath)
	cmd.Env = append(os.Environ(), "NO_COLOR=true")
	cmd.Stdin = strings.NewReader("\n")

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	_ = os.WriteFile(logPath, out.Bytes(), 0o644)
	if runCtx.Err() != nil {
		return nil, runCtx.Err()
	}
	if err != nil {
		return nil, err
	}
	return parseOutputFormats(out.String()), nil
}

func (c *Cpp2IL) runFormat(ctx context.Context, stageDir, outDir, logsDir, format string) formatResult {
	formatOut := filepath.Join(outDir, "cpp2il", safeName(format))
	logPath := filepath.Join(logsDir, safeName(format)+".log")
	args := c.args(stageDir, formatOut, format)

	start := time.Now()
	runCtx, cancel := c.withTimeout(ctx)
	defer cancel()

	cmd := exec.CommandContext(runCtx, c.BinPath, args...)
	cmd.Dir = filepath.Dir(c.BinPath)
	cmd.Env = append(os.Environ(), "NO_COLOR=true")
	cmd.Stdin = strings.NewReader("\n")

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	_ = os.WriteFile(logPath, out.Bytes(), 0o644)

	result := formatResult{
		Name:           format,
		OutputDir:      fsutil.MustRel(outDir, formatOut),
		Log:            fsutil.MustRel(outDir, logPath),
		Args:           args,
		DurationMillis: time.Since(start).Milliseconds(),
	}
	if runCtx.Err() != nil {
		result.Error = runCtx.Err().Error()
		return result
	}
	if err != nil {
		result.Error = err.Error()
		if exitErr := new(exec.ExitError); errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		}
		return result
	}
	return result
}

func (c *Cpp2IL) args(stageDir, outDir, format string) []string {
	args := []string{
		"--game-path=" + stageDir,
		"--exe-name=" + defaultExeName,
		"--output-to=" + outDir,
		"--output-as=" + format,
	}
	if c.Verbose {
		args = append(args, "--verbose")
	}
	for _, p := range c.Processors {
		p = strings.TrimSpace(p)
		if p != "" {
			args = append(args, "--use-processor="+p)
		}
	}
	args = append(args, c.ExtraArgs...)
	return args
}

func (c *Cpp2IL) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.Timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, c.Timeout)
}

func (c *Cpp2IL) stageInput(in Input, stageDir string) error {
	if err := os.RemoveAll(stageDir); err != nil {
		return err
	}
	dataDir := filepath.Join(stageDir, defaultExeName+"_Data")
	metadataDir := filepath.Join(dataDir, "il2cpp_data", "Metadata")
	if err := os.MkdirAll(metadataDir, 0o755); err != nil {
		return err
	}

	// Cpp2IL's game-path flow expects an executable-style root. The placeholder
	// gives it a stable exe name while the real code lives in GameAssembly.
	for _, name := range []string{defaultExeName, defaultExeName + ".exe"} {
		if err := os.WriteFile(filepath.Join(stageDir, name), []byte{}, 0o755); err != nil {
			return err
		}
	}

	if err := linkOrCopyFile(in.GameAssembly, filepath.Join(stageDir, filepath.Base(in.GameAssembly))); err != nil {
		return fmt.Errorf("staging GameAssembly: %w", err)
	}
	if err := linkOrCopyFile(in.Metadata, filepath.Join(metadataDir, "global-metadata.dat")); err != nil {
		return fmt.Errorf("staging metadata: %w", err)
	}

	for _, name := range []string{"globalgamemanagers", "globalgamemanagers.assets"} {
		src := filepath.Join(in.DataDir, name)
		if fsutil.Exists(src) {
			if err := linkOrCopyFile(src, filepath.Join(dataDir, name)); err != nil {
				return fmt.Errorf("staging %s: %w", name, err)
			}
		}
	}
	return nil
}

func linkOrCopyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	absSrc, err := filepath.Abs(src)
	if err != nil {
		absSrc = src
	}
	if err := os.Symlink(absSrc, dst); err == nil {
		return nil
	}
	return fsutil.CopyFile(absSrc, dst)
}

// ResolveCpp2ILBinary returns the configured Cpp2IL executable. The first
// argument accepts either a directory containing the native binary or the binary
// path itself, matching the common "tools/il2cpp/cpp2il" layout.
func ResolveCpp2ILBinary(dir, configured string) string {
	if configured != "" {
		return absIfExists(configured)
	}
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		return absIfExists(dir)
	}
	var names []string
	switch runtime.GOOS {
	case "windows":
		names = []string{"Cpp2IL-Win.exe", "Cpp2IL.exe", "cpp2il.exe"}
	case "darwin":
		names = []string{"Cpp2IL-Mac", "Cpp2IL", "cpp2il"}
	default:
		names = []string{"Cpp2IL-Linux", "Cpp2IL", "cpp2il"}
	}
	for _, name := range names {
		p := filepath.Join(dir, name)
		if fsutil.Exists(p) {
			return absIfExists(p)
		}
	}
	return filepath.Join(dir, names[0])
}

func absIfExists(path string) string {
	if !fsutil.Exists(path) {
		return path
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func cleanFormats(formats []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range formats {
		f = strings.TrimSpace(f)
		if f == "" || seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, f)
	}
	return out
}

func parseOutputFormats(text string) []string {
	seen := map[string]bool{}
	var formats []string
	inList := false
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(stripANSI(line))
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "available output formats") {
			inList = true
			continue
		}
		if !inList && !strings.Contains(lower, "id:") {
			continue
		}
		if strings.Contains(lower, "output format") || strings.Contains(lower, "available") || strings.Contains(lower, "name") {
			if !strings.Contains(lower, "id:") {
				continue
			}
		}

		candidate := formatIDFromLine(line)
		if candidate == "" || isLogLevel(candidate) || !isFormatName(candidate) || seen[candidate] {
			continue
		}
		seen[candidate] = true
		formats = append(formats, candidate)
	}
	return formats
}

func formatIDFromLine(line string) string {
	lower := strings.ToLower(line)
	if idx := strings.Index(lower, "id:"); idx >= 0 {
		rest := strings.TrimSpace(line[idx+len("id:"):])
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			return ""
		}
		return strings.Trim(fields[0], "[](),")
	}

	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(strings.TrimSuffix(fields[0], ":"), "[](),")
}

func isLogLevel(s string) bool {
	switch strings.ToUpper(s) {
	case "VERB", "DEBUG", "INFO", "WARN", "WARNING", "FAIL", "ERROR":
		return true
	default:
		return false
	}
}

func isFormatName(s string) bool {
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '.' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) && (s[i] < '@' || s[i] > '~') {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func safeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	if b.Len() == 0 {
		return "default"
	}
	return b.String()
}

func writeManifest(path string, m manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), fs.FileMode(0o644))
}

type manifest struct {
	Tool       string         `json:"tool"`
	Binary     string         `json:"binary"`
	FullDump   bool           `json:"full_dump"`
	Inputs     inputManifest  `json:"inputs"`
	Formats    []formatResult `json:"formats"`
	StartedAt  string         `json:"started_at"`
	FinishedAt string         `json:"finished_at"`
}

type inputManifest struct {
	AppPath          string `json:"app_path,omitempty"`
	DataDir          string `json:"data_dir"`
	GameAssembly     string `json:"game_assembly"`
	GameAssemblyHash string `json:"game_assembly_hash,omitempty"`
	Metadata         string `json:"metadata"`
	MetadataHash     string `json:"metadata_hash,omitempty"`
}

type formatResult struct {
	Name           string   `json:"name"`
	OutputDir      string   `json:"output_dir"`
	Log            string   `json:"log"`
	Args           []string `json:"args"`
	ExitCode       int      `json:"exit_code,omitempty"`
	DurationMillis int64    `json:"duration_ms"`
	Error          string   `json:"error,omitempty"`
}

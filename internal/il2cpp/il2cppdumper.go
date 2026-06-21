package il2cpp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"rotmg-extractor/internal/fsutil"
	"rotmg-extractor/internal/logx"
)

// outputNames are the top-level artifacts Il2CppDumper produces that we record
// in the manifest. Their presence is also how Dump confirms a successful run.
var outputNames = []string{"DummyDll", "dump.cs", "il2cpp.h", "script.json", "stringliteral.json"}

// Il2CppDumper drives Perfare's Il2CppDumper against a prepared RotMG build.
// Unlike Cpp2IL it takes the GameAssembly binary and dumpable metadata as direct
// path arguments and needs no staged game directory.
type Il2CppDumper struct {
	// BinPath is the resolved executable, or the Il2CppDumper.dll when UseDotnet
	// is set.
	BinPath string
	// UseDotnet runs BinPath via "dotnet" (the cross-platform managed build).
	UseDotnet bool
	// ExtraArgs are appended to every run.
	ExtraArgs []string
	// ForceVersion, when set, is written into config.json as ForceIl2CppVersion +
	// ForceVersion to override metadata-version auto-detection.
	ForceVersion string
	// KeepConfig leaves an existing config.json next to the binary untouched.
	KeepConfig bool
	// Timeout bounds the dump invocation. 0 disables the timeout.
	Timeout time.Duration
	Log     *logx.Logger
}

// Name identifies the backend.
func (d *Il2CppDumper) Name() string { return "il2cppdumper" }

// Available reports whether the configured Il2CppDumper binary exists.
func (d *Il2CppDumper) Available() bool {
	if d == nil || d.BinPath == "" {
		return false
	}
	info, err := os.Stat(d.BinPath)
	return err == nil && !info.IsDir()
}

// program is the executable to invoke ("dotnet" for the managed build).
func (d *Il2CppDumper) program() string {
	if d.UseDotnet {
		return "dotnet"
	}
	return d.BinPath
}

// args builds the argument vector (excluding the program name) for dumping in
// into outDir.
func (d *Il2CppDumper) args(in Input, outDir string) []string {
	var args []string
	if d.UseDotnet {
		args = append(args, d.BinPath)
	}
	args = append(args, in.GameAssembly, in.Metadata, outDir)
	args = append(args, d.ExtraArgs...)
	return args
}

// Dump runs Il2CppDumper against the prepared build, writing artifacts and a
// manifest into outDir.
func (d *Il2CppDumper) Dump(ctx context.Context, in Input, outDir string) error {
	if !d.Available() {
		return fmt.Errorf("Il2CppDumper binary not found at %q", d.BinPath)
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
	logsDir := filepath.Join(outDir, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return err
	}

	if err := d.ensureConfig(); err != nil {
		return fmt.Errorf("preparing Il2CppDumper config: %w", err)
	}

	args := d.args(in, outDir)
	m := dumperManifest{
		Tool:      d.Name(),
		Binary:    d.BinPath,
		UseDotnet: d.UseDotnet,
		Inputs: inputManifest{
			AppPath:      in.AppPath,
			DataDir:      in.DataDir,
			GameAssembly: in.GameAssembly,
			Metadata:     in.Metadata,
		},
		Args:      args,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}
	m.Inputs.GameAssemblyHash, _ = fsutil.HashFile(in.GameAssembly)
	m.Inputs.MetadataHash, _ = fsutil.HashFile(in.Metadata)

	runCtx := ctx
	var cancel context.CancelFunc
	if d.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, d.Timeout)
	} else {
		runCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	if d.Log != nil {
		d.Log.Info("Running Il2CppDumper...")
	}
	start := time.Now()
	cmd := exec.CommandContext(runCtx, d.program(), args...)
	cmd.Dir = filepath.Dir(d.BinPath)
	cmd.Env = append(os.Environ(), "NO_COLOR=true")
	cmd.Stdin = strings.NewReader("\n")

	out, runErr := cmd.CombinedOutput()
	_ = os.WriteFile(filepath.Join(logsDir, "il2cppdumper.log"), out, 0o644)

	m.DurationMillis = time.Since(start).Milliseconds()
	m.Outputs = collectOutputs(outDir)
	m.FinishedAt = time.Now().UTC().Format(time.RFC3339)

	if ctxErr := runCtx.Err(); ctxErr != nil {
		m.Error = ctxErr.Error()
	} else if runErr != nil {
		m.Error = runErr.Error()
		if exitErr := new(exec.ExitError); errors.As(runErr, &exitErr) {
			m.ExitCode = exitErr.ExitCode()
		}
	}

	if writeErr := writeDumperManifest(filepath.Join(outDir, "manifest.json"), m); writeErr != nil {
		return writeErr
	}

	if m.Error != "" {
		return fmt.Errorf("Il2CppDumper failed: %s", m.Error)
	}
	if len(m.Outputs) == 0 {
		return errors.New("Il2CppDumper produced no recognized output")
	}
	return nil
}

// ensureConfig writes a config.json next to the binary so Il2CppDumper runs
// non-interactively (RequireAnyKey=false), preserving any existing options. When
// KeepConfig is set and a config already exists, it is left untouched.
func (d *Il2CppDumper) ensureConfig() error {
	path := filepath.Join(filepath.Dir(d.BinPath), "config.json")

	cfg := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		if d.KeepConfig {
			return nil
		}
		_ = json.Unmarshal(data, &cfg) // best-effort; rewrite cleanly on parse failure
	} else if !os.IsNotExist(err) {
		return err
	}

	cfg["RequireAnyKey"] = false
	if d.ForceVersion != "" {
		cfg["ForceIl2CppVersion"] = true
		if v, err := strconv.ParseFloat(d.ForceVersion, 64); err == nil {
			cfg["ForceVersion"] = v
		} else {
			cfg["ForceVersion"] = d.ForceVersion
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// collectOutputs returns the recognized Il2CppDumper artifacts present in outDir,
// as names relative to outDir.
func collectOutputs(outDir string) []string {
	var found []string
	for _, name := range outputNames {
		if fsutil.Exists(filepath.Join(outDir, name)) {
			found = append(found, name)
		}
	}
	return found
}

// ResolveIl2CppDumperBinary returns the configured Il2CppDumper executable and
// whether it must be run via "dotnet" (the cross-platform Il2CppDumper.dll). The
// dir argument accepts either a directory containing the binary or the binary
// path itself, matching the "tools/il2cpp/il2cppdumper" layout.
func ResolveIl2CppDumperBinary(dir, configured string) (string, bool) {
	if configured != "" {
		return absIfExists(configured), isDotnetDLL(configured)
	}
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		return absIfExists(dir), isDotnetDLL(dir)
	}
	var names []string
	switch runtime.GOOS {
	case "windows":
		names = []string{"Il2CppDumper.exe", "Il2CppDumper-win.exe"}
	case "darwin":
		names = []string{"Il2CppDumper-mac", "Il2CppDumper"}
	default:
		names = []string{"Il2CppDumper-linux", "Il2CppDumper"}
	}
	for _, name := range names {
		p := filepath.Join(dir, name)
		if fsutil.Exists(p) {
			return absIfExists(p), false
		}
	}
	// Fall back to the cross-platform managed build if present.
	if dll := filepath.Join(dir, "Il2CppDumper.dll"); fsutil.Exists(dll) {
		return absIfExists(dll), true
	}
	return filepath.Join(dir, names[0]), false
}

func isDotnetDLL(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".dll")
}

type dumperManifest struct {
	Tool           string        `json:"tool"`
	Binary         string        `json:"binary"`
	UseDotnet      bool          `json:"use_dotnet"`
	Inputs         inputManifest `json:"inputs"`
	Args           []string      `json:"args"`
	Outputs        []string      `json:"outputs"`
	ExitCode       int           `json:"exit_code,omitempty"`
	DurationMillis int64         `json:"duration_ms"`
	Error          string        `json:"error,omitempty"`
	StartedAt      string        `json:"started_at"`
	FinishedAt     string        `json:"finished_at"`
}

func writeDumperManifest(path string, m dumperManifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// Package localsrc locates a RotMG build that is already installed on the local
// system and resolves it into the components the extractor needs: the Unity
// Data directory, the il2cpp metadata, and the GameAssembly binary.
//
// It understands the macOS .app bundle layout and a plain "*_Data" directory
// layout (Windows/Linux style), so the same code path works across platforms.
package localsrc

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"rotmg-extractor/internal/fsutil"
)

// Build describes an installed build on disk.
type Build struct {
	// AppPath is the install root supplied by the user (an .app bundle or a
	// directory containing a *_Data folder).
	AppPath string
	// DataDir is the Unity data directory (contains globalgamemanagers,
	// level*, resources.assets, il2cpp_data, ...).
	DataDir string
	// Metadata is the path to global-metadata.dat.
	Metadata string
	// GameAssembly is the il2cpp game assembly (.dylib/.so/.dll), or "" if not
	// found alongside the build.
	GameAssembly string
	// Hash is a short content hash of the metadata, used as the build identity
	// for new-build detection.
	Hash string
}

// gameAssemblyNames lists the platform-specific il2cpp binary names.
var gameAssemblyNames = []string{
	"GameAssembly.dylib", // macOS
	"GameAssembly.so",    // Linux
	"GameAssembly.dll",   // Windows
}

// unityPlayerNames lists the platform-specific Unity runtime library names.
var unityPlayerNames = []string{
	"UnityPlayer.dylib",
	"UnityPlayer.so",
	"UnityPlayer.dll",
}

// NativeFiles returns the native binaries and metadata worth archiving and
// scanning for the build version: global-metadata.dat, the il2cpp GameAssembly,
// and the UnityPlayer runtime (when present). The metadata is listed first
// because it is where the version string lives.
func (b Build) NativeFiles() []string {
	var files []string
	if b.Metadata != "" {
		files = append(files, b.Metadata)
	}
	if b.GameAssembly != "" {
		files = append(files, b.GameAssembly)
		dir := filepath.Dir(b.GameAssembly)
		for _, name := range unityPlayerNames {
			p := filepath.Join(dir, name)
			if fsutil.Exists(p) {
				files = append(files, p)
			}
		}
	}
	return files
}

// Discover returns the install path to use. If configured is non-empty it is
// used directly; otherwise OS-specific default locations are tried in order.
// The returned error lists the paths tried so the user knows what to configure.
func Discover(configured string) (string, error) {
	var candidates []string
	if configured != "" {
		candidates = []string{configured}
	} else {
		candidates = DefaultCandidates()
	}

	var tried []string
	for _, c := range candidates {
		if c == "" {
			continue
		}
		tried = append(tried, c)
		if fsutil.Exists(c) {
			return c, nil
		}
	}
	return "", fmt.Errorf("no RotMG install found (set source.local_path); tried: %v", tried)
}

// DefaultCandidates lists best-guess install locations for the current OS.
// These mirror the DECA launcher's install layout. The Windows and Linux paths
// are best-effort guesses; override with source.local_path if they are wrong.
func DefaultCandidates() []string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return []string{
			filepath.Join(home, ".local", "share", "RealmOfTheMadGod", "Production", "RotMGExalt.app"),
		}
	case "windows":
		var c []string
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			c = append(c, filepath.Join(local, "RealmOfTheMadGod", "Production", "RotMGExalt"))
		}
		c = append(c, filepath.Join(home, ".local", "share", "RealmOfTheMadGod", "Production", "RotMGExalt"))
		return c
	default: // linux and others
		return []string{
			filepath.Join(home, ".local", "share", "RealmOfTheMadGod", "Production", "RotMGExalt"),
		}
	}
}

// Locate inspects appPath and resolves it into a Build. It returns an error if
// the path does not look like a Unity il2cpp install (no Data dir or metadata).
func Locate(appPath string) (Build, error) {
	appPath = filepath.Clean(appPath)
	info, err := os.Stat(appPath)
	if err != nil {
		return Build{}, fmt.Errorf("install path %q: %w", appPath, err)
	}
	if !info.IsDir() {
		return Build{}, fmt.Errorf("install path %q is not a directory", appPath)
	}

	// A downloaded macOS build lands as "<dir>/RotMGExalt.app/..."; descend into
	// the bundle so the macOS layout resolution below applies.
	if !hasSuffix(appPath, ".app") {
		if app := findAppBundle(appPath); app != "" {
			appPath = app
		}
	}

	dataDir, err := findDataDir(appPath)
	if err != nil {
		return Build{}, err
	}

	metadata := filepath.Join(dataDir, "il2cpp_data", "Metadata", "global-metadata.dat")
	if !fsutil.Exists(metadata) {
		return Build{}, fmt.Errorf("global-metadata.dat not found under %q", dataDir)
	}

	hash, err := fsutil.HashFile(metadata)
	if err != nil {
		return Build{}, fmt.Errorf("hashing metadata: %w", err)
	}

	return Build{
		AppPath:      appPath,
		DataDir:      dataDir,
		Metadata:     metadata,
		GameAssembly: findGameAssembly(appPath, dataDir),
		Hash:         hash,
	}, nil
}

// findAppBundle returns the first "*.app" subdirectory of dir, or "".
func findAppBundle(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() && filepath.Ext(e.Name()) == ".app" {
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
}

// findDataDir resolves the Unity Data directory for the various install layouts.
func findDataDir(appPath string) (string, error) {
	// macOS .app bundle: Contents/Resources/Data
	macData := filepath.Join(appPath, "Contents", "Resources", "Data")
	if fsutil.Exists(filepath.Join(macData, "il2cpp_data")) {
		return macData, nil
	}

	// Windows/Linux: a sibling "*_Data" directory, or the path is itself a Data dir.
	if fsutil.Exists(filepath.Join(appPath, "il2cpp_data")) {
		return appPath, nil
	}
	entries, err := os.ReadDir(appPath)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() && (filepath.Ext(e.Name()) == "" && hasSuffix(e.Name(), "_Data")) {
			candidate := filepath.Join(appPath, e.Name())
			if fsutil.Exists(filepath.Join(candidate, "il2cpp_data")) {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("could not find a Unity Data directory under %q", appPath)
}

// findGameAssembly searches the common locations for the il2cpp binary.
func findGameAssembly(appPath, dataDir string) string {
	candidates := []string{
		filepath.Join(appPath, "Contents", "Frameworks"), // macOS
		appPath,               // Windows (next to the exe)
		filepath.Dir(dataDir), // sibling of *_Data
	}
	for _, dir := range candidates {
		for _, name := range gameAssemblyNames {
			p := filepath.Join(dir, name)
			if fsutil.Exists(p) {
				return p
			}
		}
	}
	return ""
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

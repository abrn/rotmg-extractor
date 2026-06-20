// Package config defines the application configuration and loads it from a YAML
// file, applying defaults for any unset fields.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level application configuration.
type Config struct {
	Source      Source      `yaml:"source"`
	Build       Build       `yaml:"build"`
	Extraction  Extraction  `yaml:"extraction"`
	AssetRipper AssetRipper `yaml:"assetripper"`
	Notify      Notify      `yaml:"notify"`
	Poll        Poll        `yaml:"poll"`
	Output      Output      `yaml:"output"`
	Logging     Logging     `yaml:"logging"`
}

// Notify configures new-build notifications.
type Notify struct {
	Discord DiscordNotify `yaml:"discord"`
}

// DiscordNotify configures the Discord webhook notifier.
type DiscordNotify struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
	RoleID     string `yaml:"role_id"`
}

// Build holds build-level overrides.
type Build struct {
	// VersionOverride sets the Exalt version when it can't be auto-detected
	// (newer builds no longer embed it where the old extractor looked).
	VersionOverride string `yaml:"version_override"`
}

// Extraction selects the Unity asset extraction backend.
type Extraction struct {
	// Backend is "native" (pure-Go TextAsset parser, no external binary) or
	// "assetripper" (full asset export via the bundled AssetRipper binary).
	Backend string `yaml:"backend"`
	// DecryptMetadata, when true, produces a decrypted global-metadata.dat for
	// il2cpp dumping. Automatically skipped when the metadata is already valid
	// (the macOS build ships it decrypted). On by default.
	DecryptMetadata bool `yaml:"decrypt_metadata"`
}

// AssetRipper configures the bundled Unity asset extractor.
type AssetRipper struct {
	// Dir is the directory holding the AssetRipper binary. The OS-specific
	// executable name (AssetRipper.GUI.Free[.exe]) is resolved within it.
	Dir string `yaml:"dir"`
	// Port is the local port AssetRipper hosts its API on. 0 picks a free port.
	Port int `yaml:"port"`
	// Export is "primary" (assets only) or "project" (full Unity project).
	Export string `yaml:"export"`
}

// Source selects where builds come from. Stage 1 supported remote downloads;
// while the live endpoints are being re-discovered, "local" mode extracts a
// build already installed on disk.
type Source struct {
	// Mode is "local" or "remote".
	Mode string `yaml:"mode"`
	// Platforms lists which platforms to watch/download in remote mode
	// ("windows", "macos"). Defaults to Windows only.
	Platforms []string `yaml:"platforms"`
	// LocalPath is the installed build root for local mode (an .app bundle on
	// macOS, or a directory containing a *_Data folder on Windows/Linux). If
	// empty, the install is auto-discovered from OS-specific default locations.
	LocalPath string `yaml:"local_path"`
	// Snapshot, when true, copies the full build files (the whole Data dir,
	// hundreds of MB) into the output tree. Off by default.
	Snapshot bool `yaml:"snapshot"`
	// CopyGameFiles, when true, copies the native game binaries
	// (GameAssembly/UnityPlayer) and global-metadata.dat into the output. These
	// are also scanned for the build version. On by default (~130 MB).
	CopyGameFiles bool `yaml:"copy_game_files"`
	// FullDownload, when true, downloads every file in the build manifest
	// instead of only the essential ones (binaries, metadata, Unity
	// SerializedFiles). Off by default — the skipped texture/audio streaming
	// data is ~80% of a build.
	FullDownload bool `yaml:"full_download"`
	// Incremental keeps a persistent copy of the build files and only
	// re-downloads files whose checksum changed between builds. Off by default:
	// it saves bandwidth (unchanged binaries aren't re-fetched) at the cost of
	// keeping the build files on disk (~500 MB/platform, decompressed).
	Incremental bool `yaml:"incremental"`
}

// Poll controls how often the extractor checks for new builds.
type Poll struct {
	// ClientCheckDelayMinutes is the delay between full polling passes.
	// (The original config split client/launcher delays; Stage 1 uses a single
	// loop delay and keeps both fields for forward compatibility.)
	ClientCheckDelayMinutes   int `yaml:"client_check_delay_minutes"`
	LauncherCheckDelayMinutes int `yaml:"launcher_check_delay_minutes"`
}

// Output controls where build files are written.
type Output struct {
	// Dir is the root output directory (contains temp/ and publish/).
	Dir string `yaml:"dir"`
	// KeepBuilds bounds how many versioned builds to retain per platform/build
	// type; older ones are pruned after publishing. 0 keeps all.
	KeepBuilds int `yaml:"keep_builds"`
}

// Logging controls log verbosity and destinations.
type Logging struct {
	Level   string `yaml:"level"`
	Console bool   `yaml:"console"`
	Colors  bool   `yaml:"colors"`
	File    bool   `yaml:"file"`
}

// Default returns a Config populated with sensible defaults.
func Default() Config {
	return Config{
		Source: Source{
			Mode:          "local",
			Platforms:     []string{"windows"},
			LocalPath:     "", // empty => auto-discover per OS
			Snapshot:      false,
			CopyGameFiles: true,
		},
		Build: Build{
			VersionOverride: "", // auto-detected from metadata; set only as a fallback
		},
		Extraction: Extraction{
			Backend:         "native",
			DecryptMetadata: true,
		},
		AssetRipper: AssetRipper{
			Dir:    "tools/assetripper",
			Port:   0, // 0 => pick a free port automatically
			Export: "primary",
		},
		Notify: Notify{
			Discord: DiscordNotify{
				Enabled:    false,
				WebhookURL: "",
				RoleID:     "",
			},
		},
		Poll: Poll{
			ClientCheckDelayMinutes:   5,
			LauncherCheckDelayMinutes: 30,
		},
		Output: Output{
			Dir: "./output",
		},
		Logging: Logging{
			Level:   "debug",
			Console: true,
			Colors:  true,
			File:    true,
		},
	}
}

// Load reads configuration from path, overlaying it onto the defaults. A
// missing file is not an error: the defaults are returned.
func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("reading config %q: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config %q: %w", path, err)
	}
	return cfg, nil
}

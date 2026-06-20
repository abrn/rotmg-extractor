// Package pipeline orchestrates the per-build extraction process: checking for
// new builds, downloading, extracting and publishing.
//
// Stage 1 implements the control flow and the "is this a new build?" check.
// Downloading, extraction and publishing are stubbed and will be filled in by
// later stages.
package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"rotmg-extractor/internal/extract"
	"rotmg-extractor/internal/fsutil"
	"rotmg-extractor/internal/localsrc"
	"rotmg-extractor/internal/logx"
	"rotmg-extractor/internal/paths"
	"rotmg-extractor/internal/rotmg"
)

// Extractor extracts Unity assets from a build's Data directory into outDir.
// Implemented by the AssetRipper client and the native TextAsset parser.
type Extractor interface {
	Extract(ctx context.Context, dataDir, outDir string) error
	Available() bool
	Name() string
}

// Pipeline carries the dependencies shared across build runs.
type Pipeline struct {
	Log    *logx.Logger
	Layout paths.Layout
	// Extractor extracts Unity assets. May be nil to skip extraction.
	Extractor Extractor
	// VersionOverride is used as the Exalt version when it can't be detected.
	VersionOverride string
}

// New constructs a Pipeline. Set Extractor and VersionOverride on the result as
// needed.
func New(log *logx.Logger, layout paths.Layout) *Pipeline {
	return &Pipeline{Log: log, Layout: layout}
}

// Run executes the full extract process for a single environment + build type.
// It returns nil whether or not a new build was found; an error is returned
// only for unexpected failures.
func (p *Pipeline) Run(ctx context.Context, env rotmg.Environment, settings rotmg.AppSettings, bt rotmg.BuildType) error {
	build := settings.Build(bt)

	filesDir := p.Layout.FilesDir(env.Name, string(bt))
	workDir := p.Layout.WorkDir(env.Name, string(bt))
	publishDir := p.Layout.PublishDir(env.Name, string(bt))

	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return fmt.Errorf("creating work dir: %w", err)
	}

	// Attach a per-build log file alongside the console output.
	if err := p.Log.SetFile(filepath.Join(workDir, "log.txt")); err != nil {
		p.Log.Warn("could not open build log file: %v", err)
	}
	defer p.Log.SetFile("")

	p.Log.PrintTime()
	p.Log.Info("Starting %s %s", env.Name, bt)
	p.Log.Indent()
	defer p.Log.Dedent()

	isNew, err := p.preBuildSetup(env, bt, build, workDir, publishDir)
	if err != nil {
		return err
	}
	if !isNew {
		return nil
	}

	// --- Stages 2-5: implemented incrementally ---
	if err := p.downloadBuild(ctx, env, bt, build, filesDir, workDir); err != nil {
		return err
	}
	if err := p.extractBuild(ctx, bt, filesDir, workDir); err != nil {
		return err
	}
	if err := p.outputBuild(ctx, env, bt, build, workDir, publishDir); err != nil {
		return err
	}

	p.Log.Info("Done %s %s", env.Name, bt)
	return nil
}

// preBuildSetup asserts a build exists and is newer than the published one,
// then records the new hash/version into the work dir. It returns true when the
// build is new and processing should continue.
func (p *Pipeline) preBuildSetup(env rotmg.Environment, bt rotmg.BuildType, build rotmg.BuildInfo, workDir, publishDir string) (bool, error) {
	if !build.Available() {
		p.Log.Warn("%s does not have a %s build available, aborting.", env.Name, bt)
		return false, nil
	}

	// Compare against the currently published build hash, if any.
	currentHashFile := filepath.Join(publishDir, "current", "build_hash.txt")
	if data, err := os.ReadFile(currentHashFile); err == nil {
		if strings.TrimSpace(string(data)) == build.BuildHash {
			p.Log.Info("Current build hash is equal, aborting.")
			return false, nil
		}
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("reading current build hash: %w", err)
	}

	p.Log.Info("New build! Build hash: %s", build.BuildHash)
	if err := writeFile(filepath.Join(workDir, "build_hash.txt"), build.BuildHash); err != nil {
		return false, err
	}
	if err := writeFile(filepath.Join(workDir, "build_version.txt"), build.BuildVersion); err != nil {
		return false, err
	}
	return true, nil
}

// downloadBuild downloads (and unpacks/archives) the build files. Stage 2.
func (p *Pipeline) downloadBuild(ctx context.Context, env rotmg.Environment, bt rotmg.BuildType, build rotmg.BuildInfo, filesDir, workDir string) error {
	p.Log.Info("Build URL is %s", build.BuildURL())
	p.Log.Warn("download not yet implemented (Stage 2)")
	return nil
}

// extractBuild extracts Unity assets and dumps il2cpp. Stages 3 & 5.
func (p *Pipeline) extractBuild(ctx context.Context, bt rotmg.BuildType, filesDir, workDir string) error {
	p.Log.Warn("extraction not yet implemented (Stages 3 & 5)")
	return nil
}

// outputBuild publishes the build and sends notifications. Stage 4.
func (p *Pipeline) outputBuild(ctx context.Context, env rotmg.Environment, bt rotmg.BuildType, build rotmg.BuildInfo, workDir, publishDir string) error {
	p.Log.Warn("publishing not yet implemented (Stage 4)")
	return nil
}

// RunLocal executes the extract process against a build already installed on
// the local system, bypassing the live download flow. The build is always a
// Client build; envName is used only for output directory naming.
//
// Extraction reads directly from the source install — the build is not copied
// unless Snapshot is requested — and only proceeds when the build is new.
func (p *Pipeline) RunLocal(ctx context.Context, envName string, build localsrc.Build, snapshot bool) error {
	const buildType = "Client"

	filesDir := p.Layout.FilesDir(envName, buildType)
	workDir := p.Layout.WorkDir(envName, buildType)
	publishDir := p.Layout.PublishDir(envName, buildType)

	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return fmt.Errorf("creating work dir: %w", err)
	}
	if err := p.Log.SetFile(filepath.Join(workDir, "log.txt")); err != nil {
		p.Log.Warn("could not open build log file: %v", err)
	}
	defer p.Log.SetFile("")

	p.Log.PrintTime()
	p.Log.Info("Starting %s %s (local: %s)", envName, buildType, build.AppPath)
	p.Log.Indent()
	defer p.Log.Dedent()

	// New-build detection: skip everything (including any copy) when the build
	// identity matches what was last processed.
	if !p.isNewBuild(build.Hash, publishDir) {
		p.Log.Info("Build hash unchanged (%s) - nothing to do.", build.Hash)
		return nil
	}
	p.Log.Info("New build! Build hash: %s", build.Hash)
	if err := writeFile(filepath.Join(workDir, "build_hash.txt"), build.Hash); err != nil {
		return err
	}

	// Optionally snapshot the original build files (off by default).
	if snapshot {
		if err := p.collectLocalBuild(build, filesDir); err != nil {
			return err
		}
	}

	if err := p.extractLocalBuild(ctx, build, workDir); err != nil {
		return err
	}

	// Mark this build as processed so subsequent runs skip it.
	if err := p.markProcessed(build.Hash, publishDir); err != nil {
		return err
	}

	p.Log.Info("Done %s %s", envName, buildType)
	return nil
}

// processedHashFile is the persistent marker recording the last build processed
// for an env/build. It survives temp-dir clearing because it lives under
// publish/, and it is the same path the remote flow compares against.
func processedHashFile(publishDir string) string {
	return filepath.Join(publishDir, "current", "build_hash.txt")
}

// isNewBuild reports whether hash differs from the last processed build.
func (p *Pipeline) isNewBuild(hash, publishDir string) bool {
	data, err := os.ReadFile(processedHashFile(publishDir))
	if err != nil {
		return true // no marker yet (or unreadable) => treat as new
	}
	return strings.TrimSpace(string(data)) != hash
}

// markProcessed persists the build identity after a successful extraction.
func (p *Pipeline) markProcessed(hash, publishDir string) error {
	return writeFile(processedHashFile(publishDir), hash)
}

// collectLocalBuild copies the installed game files into the build files dir so
// the build is snapshotted in its original state.
func (p *Pipeline) collectLocalBuild(build localsrc.Build, filesDir string) error {
	p.Log.Info("Snapshotting game files...")
	p.Log.Indent()
	defer p.Log.Dedent()

	if err := os.RemoveAll(filesDir); err != nil {
		return fmt.Errorf("clearing files dir: %w", err)
	}

	// Copy the Unity Data directory (preserving its folder name).
	dataDst := filepath.Join(filesDir, filepath.Base(build.DataDir))
	p.Log.Info("Copying %s -> %s", build.DataDir, dataDst)
	if err := fsutil.CopyDir(build.DataDir, dataDst); err != nil {
		return fmt.Errorf("copying data dir: %w", err)
	}

	// Copy the game assembly binary, if present.
	if build.GameAssembly != "" {
		gaDst := filepath.Join(filesDir, filepath.Base(build.GameAssembly))
		p.Log.Info("Copying %s -> %s", build.GameAssembly, gaDst)
		if err := fsutil.CopyFile(build.GameAssembly, gaDst); err != nil {
			return fmt.Errorf("copying game assembly: %w", err)
		}
	} else {
		p.Log.Warn("No GameAssembly binary found - il2cpp dump will be skipped")
	}

	p.Log.Info("Game files snapshotted")
	return nil
}

// extractLocalBuild extracts data straight from the source build. Currently:
// the Exalt version (best-effort) and Unity assets via the configured backend.
func (p *Pipeline) extractLocalBuild(ctx context.Context, build localsrc.Build, workDir string) error {
	p.Log.Info("Extracting build...")
	p.Log.Indent()
	defer p.Log.Dedent()

	version, err := extract.ExaltVersion(build.Metadata)
	if err != nil {
		return fmt.Errorf("extracting exalt version: %w", err)
	}
	switch {
	case version != "":
		p.Log.Info("Exalt version is %q", version)
	case p.VersionOverride != "":
		version = p.VersionOverride
		p.Log.Info("Exalt version not auto-detected; using configured override %q", version)
	default:
		p.Log.Warn("Could not determine Exalt version - leaving blank")
	}
	if err := writeFile(filepath.Join(workDir, "exalt_version.txt"), version); err != nil {
		return err
	}

	// Extract Unity assets using the configured backend, reading from the source
	// Data directory directly (no copy required).
	if p.Extractor != nil && p.Extractor.Available() {
		extractedDir := filepath.Join(workDir, "extracted_assets")
		p.Log.Info("Extracting Unity assets (%s backend)...", p.Extractor.Name())
		p.Log.Indent()
		err := p.Extractor.Extract(ctx, build.DataDir, extractedDir)
		p.Log.Dedent()
		if err != nil {
			return fmt.Errorf("asset extraction: %w", err)
		}
	} else {
		p.Log.Warn("No asset extractor available - skipping Unity asset extraction")
	}

	p.Log.Warn("il2cpp dump not yet implemented (pending Il2CppInspector integration)")
	return nil
}

func writeFile(path, contents string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

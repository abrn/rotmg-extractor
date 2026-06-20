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
	"strconv"
	"strings"
	"time"

	"rotmg-extractor/internal/builddiff"
	"rotmg-extractor/internal/extract"
	"rotmg-extractor/internal/fsutil"
	"rotmg-extractor/internal/gamediff"
	"rotmg-extractor/internal/localsrc"
	"rotmg-extractor/internal/logx"
	"rotmg-extractor/internal/mergexml"
	"rotmg-extractor/internal/notify"
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
	// Notifier announces new builds. May be nil to disable notifications.
	Notifier notify.Notifier
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

	p.Log.Success("New build! Build hash: %s", build.BuildHash)
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
func (p *Pipeline) RunLocal(ctx context.Context, envName string, build localsrc.Build, snapshot, copyGameFiles bool) error {
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
		p.Log.Success("Build hash unchanged (%s) - nothing to do.", build.Hash)
		return nil
	}
	p.Log.Success("New build! Build hash: %s", build.Hash)
	if err := writeFile(filepath.Join(workDir, "build_hash.txt"), build.Hash); err != nil {
		return err
	}

	// Copy the native game binaries + metadata into the output (on by default).
	if copyGameFiles {
		if err := p.collectGameFiles(build, filepath.Join(workDir, "game_files")); err != nil {
			return err
		}
	}

	// Optionally snapshot the entire Data dir (off by default).
	if snapshot {
		if err := p.collectLocalBuild(build, filesDir); err != nil {
			return err
		}
	}

	version, err := p.extractLocalBuild(ctx, build, workDir)
	if err != nil {
		return err
	}

	// Publish the output (diff vs. previous, archive, refresh current). This
	// also persists build_hash.txt into publish/current, which drives the
	// new-build check on subsequent runs.
	fileDiff, gameSummary, err := p.publishBuild(workDir, publishDir, version, build.Hash)
	if err != nil {
		return err
	}

	p.sendNotification(ctx, envName, buildType, version, build.Hash, fileDiff, gameSummary)

	p.Log.Success("Done (%s %s)", envName, buildType)
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

// collectGameFiles copies the native binaries and metadata into destDir so the
// build's il2cpp artifacts are archived alongside the extracted assets.
func (p *Pipeline) collectGameFiles(build localsrc.Build, destDir string) error {
	files := build.NativeFiles()
	if len(files) == 0 {
		p.Log.Warn("No native game files found to copy")
		return nil
	}

	p.Log.Info("Copying game files (binaries + metadata)...")
	p.Log.Indent()
	defer p.Log.Dedent()

	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("clearing game_files dir: %w", err)
	}
	for _, src := range files {
		dst := filepath.Join(destDir, filepath.Base(src))
		p.Log.Info("Copying %s", filepath.Base(src))
		if err := fsutil.CopyFile(src, dst); err != nil {
			return fmt.Errorf("copying %s: %w", filepath.Base(src), err)
		}
	}
	return nil
}

// extractLocalBuild extracts data straight from the source build (Exalt version
// plus Unity assets via the configured backend) and returns the resolved
// version string.
func (p *Pipeline) extractLocalBuild(ctx context.Context, build localsrc.Build, workDir string) (string, error) {
	p.Log.Info("Extracting build...")
	p.Log.Indent()
	defer p.Log.Dedent()

	// Scan the native files (metadata first, then binaries) for the version.
	version, err := extract.ScanVersion(build.NativeFiles()...)
	if err != nil {
		return "", fmt.Errorf("scanning for exalt version: %w", err)
	}
	switch {
	case version != "":
		p.Log.Success("Detected Exalt version %q", version)
		if p.VersionOverride != "" && p.VersionOverride != version {
			p.Log.Warn("Configured version override %q differs from detected %q - using detected", p.VersionOverride, version)
		}
	case p.VersionOverride != "":
		version = p.VersionOverride
		p.Log.Warn("Exalt version not detected; using configured override %q", version)
	default:
		p.Log.Warn("Could not determine Exalt version - leaving blank")
	}
	if err := writeFile(filepath.Join(workDir, "exalt_version.txt"), version); err != nil {
		return "", err
	}

	// Extract Unity assets using the configured backend, reading from the source
	// Data directory directly (no copy required).
	extractedDir := filepath.Join(workDir, "extracted_assets")
	if p.Extractor != nil && p.Extractor.Available() {
		p.Log.Info("Extracting Unity assets (%s backend)...", p.Extractor.Name())
		p.Log.Indent()
		err := p.Extractor.Extract(ctx, build.DataDir, extractedDir)
		p.Log.Dedent()
		if err != nil {
			return "", fmt.Errorf("asset extraction: %w", err)
		}

		// Consolidate the extracted XML into object/ground/misc files.
		if err := mergexml.Merge(p.Log, extractedDir, filepath.Join(workDir, "merged")); err != nil {
			return "", fmt.Errorf("merging xml: %w", err)
		}
	} else {
		p.Log.Warn("No asset extractor available - skipping Unity asset extraction")
	}

	p.Log.Warn("il2cpp dump not yet implemented (pending Il2CppInspector integration)")
	return version, nil
}

// publishBuild diffs the new build against the currently published one, writes a
// changelog, archives the work output to a versioned directory, and refreshes
// publish/current. It returns the diffs for notification.
func (p *Pipeline) publishBuild(workDir, publishDir, version, hash string) (builddiff.Diff, gamediff.Summary, error) {
	p.Log.Info("Publishing build...")
	p.Log.Indent()
	defer p.Log.Dedent()

	now := time.Now()
	if err := writeFile(filepath.Join(workDir, "timestamp.txt"), strconv.FormatInt(now.Unix(), 10)); err != nil {
		return builddiff.Diff{}, gamediff.Summary{}, err
	}

	currentDir := filepath.Join(publishDir, "current")

	// Diff against the previous build before it is overwritten.
	var fileDiff builddiff.Diff
	var gameSummary gamediff.Summary
	if fsutil.Exists(currentDir) {
		var err error
		fileDiff, err = builddiff.Compare(
			filepath.Join(currentDir, "extracted_assets"),
			filepath.Join(workDir, "extracted_assets"),
		)
		if err != nil {
			return builddiff.Diff{}, gamediff.Summary{}, fmt.Errorf("file diff: %w", err)
		}
		gameSummary, err = gamediff.Compare(
			filepath.Join(currentDir, "merged"),
			filepath.Join(workDir, "merged"),
		)
		if err != nil {
			return builddiff.Diff{}, gamediff.Summary{}, fmt.Errorf("game diff: %w", err)
		}
		p.logDiff(fileDiff, gameSummary)
	} else {
		p.Log.Info("No previously published build - skipping diff")
	}

	// Write the changelog into the work dir so it is archived with the build.
	changelog := gameSummary.Markdown(version, hash, now.Format("2006-01-02 15:04:05"))
	if err := writeFile(filepath.Join(workDir, "changelog.md"), changelog); err != nil {
		return builddiff.Diff{}, gamediff.Summary{}, err
	}

	// Archive to a versioned directory (kept indefinitely).
	versionDir := filepath.Join(publishDir, versionLabel(version, hash))
	if err := replaceDir(workDir, versionDir); err != nil {
		return builddiff.Diff{}, gamediff.Summary{}, fmt.Errorf("archiving build: %w", err)
	}
	p.Log.Success("Archived build to %s", versionDir)

	// Refresh publish/current.
	if err := replaceDir(workDir, currentDir); err != nil {
		return builddiff.Diff{}, gamediff.Summary{}, fmt.Errorf("updating current: %w", err)
	}
	p.Log.Info("Updated %s", currentDir)

	return fileDiff, gameSummary, nil
}

func (p *Pipeline) logDiff(d builddiff.Diff, s gamediff.Summary) {
	p.Log.Info("Files: +%d -%d ~%d  Lines: +%d -%d",
		d.NewFiles, d.DelFiles, d.ChangedFiles, d.AddedLines, d.RemovedLines)
	oa, or, oc := s.Objects.Counts()
	ga, gr, gc := s.Ground.Counts()
	p.Log.Info("Objects: +%d -%d ~%d   Ground: +%d -%d ~%d", oa, or, oc, ga, gr, gc)
}

// sendNotification builds and dispatches a new-build notification, logging but
// not failing on errors.
func (p *Pipeline) sendNotification(ctx context.Context, env, buildType, version, hash string, d builddiff.Diff, s gamediff.Summary) {
	if p.Notifier == nil {
		return
	}
	oa, or, oc := s.Objects.Counts()
	ga, gr, gc := s.Ground.Counts()
	n := notify.Notification{
		Env: env, BuildType: buildType, Version: version, Hash: hash,
		NewFiles: d.NewFiles, DelFiles: d.DelFiles,
		AddedLines: d.AddedLines, RemovedLines: d.RemovedLines,
		ObjAdded: oa, ObjRemoved: or, ObjChanged: oc,
		GndAdded: ga, GndRemoved: gr, GndChanged: gc,
	}
	if err := p.Notifier.Notify(ctx, n); err != nil {
		p.Log.Warn("notification failed: %v", err)
	}
}

// versionLabel builds the archive directory name for a build.
func versionLabel(version, hash string) string {
	if version == "" {
		return hash
	}
	return strings.ReplaceAll(version, " ", "_") + "-" + hash
}

// replaceDir copies src over dst, removing any existing dst first.
func replaceDir(src, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	return fsutil.CopyDir(src, dst)
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

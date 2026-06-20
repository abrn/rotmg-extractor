// Command extractor polls every RotMG environment for new client and launcher
// builds and runs the extraction pipeline for each.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"rotmg-extractor/internal/assetripper"
	"rotmg-extractor/internal/config"
	"rotmg-extractor/internal/il2cpp"
	"rotmg-extractor/internal/localsrc"
	"rotmg-extractor/internal/logx"
	"rotmg-extractor/internal/notify"
	"rotmg-extractor/internal/paths"
	"rotmg-extractor/internal/pipeline"
	"rotmg-extractor/internal/rotmg"
	"rotmg-extractor/internal/unityassets"
)

func main() {
	configPath := flag.String("config", "extractor.yml", "path to the config file")
	once := flag.Bool("once", false, "run a single pass instead of looping")
	il2cppOnly := flag.Bool("il2cpp-only", false, "run only the IL2CPP dump against an existing client build")
	il2cppEnv := flag.String("il2cpp-env", "", "environment/platform for -il2cpp-only (default: local Production or first configured remote platform)")
	il2cppFormat := flag.String("il2cpp-format", "", "comma-separated Cpp2IL output format override for -il2cpp-only")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		// Logger isn't up yet; write straight to stderr and exit.
		os.Stderr.WriteString("config error: " + err.Error() + "\n")
		os.Exit(1)
	}
	if *il2cppFormat != "" {
		cfg.IL2CPP.Cpp2IL.FullDump = false
		cfg.IL2CPP.Cpp2IL.Formats = splitCSV(*il2cppFormat)
	}

	log := logx.New(logx.ParseLevel(cfg.Logging.Level), cfg.Logging.Colors)

	// Graceful shutdown on Ctrl-C / SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	layout := paths.New(cfg.Output.Dir)
	pipe := pipeline.New(log, layout)
	pipe.VersionOverride = cfg.Build.VersionOverride
	pipe.FullDownload = cfg.Source.FullDownload
	pipe.Incremental = cfg.Source.Incremental
	pipe.KeepBuilds = cfg.Output.KeepBuilds
	pipe.DecryptMetadata = cfg.Extraction.DecryptMetadata
	pipe.Extractor = buildExtractor(cfg, log)
	pipe.IL2CPPDumper = buildIL2CPPDumper(cfg, log)
	pipe.IL2CPPRequired = cfg.IL2CPP.Required
	pipe.Notifier = buildNotifier(cfg, log)
	log.Info("Using %q extraction backend", pipe.Extractor.Name())
	if pipe.IL2CPPDumper != nil {
		log.Info("Using %q il2cpp backend", pipe.IL2CPPDumper.Name())
	}

	// Clear the temp directory from any previous run.
	if err := os.RemoveAll(layout.Temp()); err != nil {
		log.Warn("could not clear temp dir: %v", err)
	}

	if *il2cppOnly {
		runIL2CPPOnly(ctx, log, pipe, cfg.Source, *il2cppEnv)
		return
	}

	delay := time.Duration(cfg.Poll.ClientCheckDelayMinutes) * time.Minute

	for {
		switch cfg.Source.Mode {
		case "local":
			runLocalPass(ctx, log, pipe, cfg.Source)
		default:
			runPass(ctx, log, pipe, cfg.Source.Platforms)
		}

		if *once {
			return
		}

		log.Info("Looping in %d minutes...\n", cfg.Poll.ClientCheckDelayMinutes)
		select {
		case <-ctx.Done():
			log.Info("Shutting down.")
			return
		case <-time.After(delay):
		}
	}
}

// buildExtractor selects the asset-extraction backend from config.
func buildExtractor(cfg config.Config, log *logx.Logger) pipeline.Extractor {
	if cfg.Extraction.Backend == "assetripper" {
		mode := assetripper.ExportPrimary
		if cfg.AssetRipper.Export == "project" {
			mode = assetripper.ExportProject
		}
		return &assetripper.Client{
			BinPath: assetripper.ResolveBinary(cfg.AssetRipper.Dir),
			Port:    cfg.AssetRipper.Port,
			Mode:    mode,
			Log:     log,
		}
	}
	// Default: the pure-Go native TextAsset extractor (cross-platform, no binary).
	return &unityassets.Extractor{Log: log}
}

// buildIL2CPPDumper constructs the configured IL2CPP dumper, or nil when
// disabled.
func buildIL2CPPDumper(cfg config.Config, log *logx.Logger) pipeline.IL2CPPDumper {
	if !cfg.IL2CPP.Enabled {
		return nil
	}
	c := cfg.IL2CPP.Cpp2IL
	return &il2cpp.Cpp2IL{
		BinPath:        il2cpp.ResolveCpp2ILBinary(c.Dir, c.Binary),
		FullDump:       c.FullDump,
		Formats:        c.Formats,
		Processors:     c.Processors,
		ExtraArgs:      c.ExtraArgs,
		Verbose:        c.Verbose,
		Timeout:        time.Duration(cfg.IL2CPP.TimeoutMinutes) * time.Minute,
		Log:            log,
		ContinueOnFail: c.ContinueOnFail,
	}
}

func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// buildNotifier constructs the configured notifier, or nil if none is enabled.
func buildNotifier(cfg config.Config, log *logx.Logger) notify.Notifier {
	d := cfg.Notify.Discord
	if d.Enabled && d.WebhookURL != "" {
		log.Info("Discord notifications enabled")
		return &notify.Discord{WebhookURL: d.WebhookURL, RoleID: d.RoleID, Log: log}
	}
	return nil
}

// runLocalPass extracts a single build installed on the local system.
func runLocalPass(ctx context.Context, log *logx.Logger, pipe *pipeline.Pipeline, src config.Source) {
	installPath, err := localsrc.Discover(src.LocalPath)
	if err != nil {
		log.Error("%v", err)
		return
	}
	build, err := localsrc.Locate(installPath)
	if err != nil {
		log.Error("Could not locate local build: %v", err)
		return
	}
	if err := pipe.RunLocal(ctx, "Production", build, src.Snapshot, src.CopyGameFiles); err != nil {
		log.Error("Local pipeline failed: %v", err)
	}
	log.Info("Done!")
}

// runIL2CPPOnly reruns Cpp2IL without the normal new-build/download/extract
// flow. Remote mode uses output/buildfiles/<env>/client, so source.incremental
// must have preserved the downloaded build files.
func runIL2CPPOnly(ctx context.Context, log *logx.Logger, pipe *pipeline.Pipeline, src config.Source, env string) {
	if pipe.IL2CPPDumper == nil {
		log.Error("IL2CPP is disabled; set il2cpp.enabled: true")
		return
	}

	var installPath string
	var err error
	if src.Mode == "local" {
		if env == "" {
			env = "Production"
		}
		installPath, err = localsrc.Discover(src.LocalPath)
	} else {
		if env == "" {
			if len(src.Platforms) > 0 {
				env = src.Platforms[0]
			} else {
				env = "windows"
			}
		}
		installPath = pipe.Layout.BuildFilesDir(env, string(rotmg.Client))
	}
	if err != nil {
		log.Error("%v", err)
		return
	}

	build, err := localsrc.Locate(installPath)
	if err != nil {
		log.Error("Could not locate IL2CPP build at %s: %v", installPath, err)
		return
	}
	if err := pipe.RunIL2CPPOnly(ctx, env, build); err != nil {
		log.Error("IL2CPP-only dump failed: %v", err)
	}
}

// runPass fetches and processes each configured platform's client and launcher.
func runPass(ctx context.Context, log *logx.Logger, pipe *pipeline.Pipeline, platforms []string) {
	if len(platforms) == 0 {
		log.Warn("No platforms configured (source.platforms) - nothing to do")
		return
	}

	for _, name := range platforms {
		if ctx.Err() != nil {
			return
		}
		platform, ok := rotmg.Platforms[name]
		if !ok {
			log.Error("Unknown platform %q (known: windows, macos) - skipping", name)
			continue
		}

		settings, err := rotmg.FetchAppSettings(ctx, platform)
		if err != nil {
			log.Error("Failed fetching app settings for %s: %v", platform.Name, err)
			continue
		}

		for _, bt := range []rotmg.BuildType{rotmg.Client, rotmg.Launcher} {
			if ctx.Err() != nil {
				return
			}
			if err := pipe.Run(ctx, platform, settings, bt); err != nil {
				log.Error("Pipeline failed for %s %s: %v", platform.Name, bt, err)
			}
		}
	}

	log.Info("Done!")
}

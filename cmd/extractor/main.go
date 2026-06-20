// Command extractor polls every RotMG environment for new client and launcher
// builds and runs the extraction pipeline for each.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rotmg-extractor/internal/assetripper"
	"rotmg-extractor/internal/config"
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
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		// Logger isn't up yet; write straight to stderr and exit.
		os.Stderr.WriteString("config error: " + err.Error() + "\n")
		os.Exit(1)
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
	pipe.Notifier = buildNotifier(cfg, log)
	log.Info("Using %q extraction backend", pipe.Extractor.Name())

	// Clear the temp directory from any previous run.
	if err := os.RemoveAll(layout.Temp()); err != nil {
		log.Warn("could not clear temp dir: %v", err)
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

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
	pipe.Extractor = buildExtractor(cfg, log)
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
			runPass(ctx, log, pipe)
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
	if err := pipe.RunLocal(ctx, "Production", build, src.Snapshot); err != nil {
		log.Error("Local pipeline failed: %v", err)
	}
	log.Info("Done!")
}

// runPass processes every environment and build type once.
func runPass(ctx context.Context, log *logx.Logger, pipe *pipeline.Pipeline) {
	for _, env := range rotmg.Environments {
		if ctx.Err() != nil {
			return
		}

		settings, err := rotmg.FetchAppSettings(ctx, env)
		if err != nil {
			log.Error("Failed fetching app settings for %s: %v", env.Name, err)
			continue
		}

		for _, bt := range []rotmg.BuildType{rotmg.Client, rotmg.Launcher} {
			if ctx.Err() != nil {
				return
			}
			if err := pipe.Run(ctx, env, settings, bt); err != nil {
				log.Error("Pipeline failed for %s %s: %v", env.Name, bt, err)
			}
		}
	}

	log.Info("Done!")
}

package main

import (
	"flag"
	"fmt"
	"log"

	"var-sync/internal/config"
	"var-sync/internal/logger"
	"var-sync/internal/sync"
	"var-sync/internal/tui"
)

const version = "1.0.0"

func main() {
	var (
		configFile  = flag.String("config", "var-sync.json", "Configuration file path")
		interactive = flag.Bool("tui", false, "Start interactive TUI mode")
		watch       = flag.Bool("watch", false, "Start file watching mode")
		showVersion = flag.Bool("version", false, "Show version")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("var-sync version %s\n", version)
		return
	}

	logger := logger.New()
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		cfg = config.New()
	}

	if cfg.LogFile != "" {
		if err := logger.SetLogFile(cfg.LogFile); err != nil {
			log.Printf("Failed to set log file: %v", err)
		}
	}

	if cfg.Debug {
		logger.SetLevel(0) // DEBUG level
	}

	if *interactive {
		app := tui.New(cfg, logger)
		if err := app.Run(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if *watch {
		syncer := sync.New(cfg, logger)
		if err := syncer.Start(); err != nil {
			log.Fatal(err)
		}
		return
	}

	flag.Usage()
}

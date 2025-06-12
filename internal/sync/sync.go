package sync

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"var-sync/internal/logger"
	"var-sync/internal/watcher"
	"var-sync/pkg/models"
)

type Syncer struct {
	config  *models.Config
	watcher *watcher.FileWatcher
	logger  *logger.Logger
}

func New(config *models.Config, logger *logger.Logger) *Syncer {
	return &Syncer{
		config: config,
		logger: logger,
	}
}

func (s *Syncer) Start() error {
	var err error
	s.watcher, err = watcher.New(s.logger)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	if err := s.watcher.SetRules(s.config.Rules); err != nil {
		return fmt.Errorf("failed to set watcher rules: %w", err)
	}

	s.logger.Info("Starting sync service with %d rules", len(s.config.Rules))

	if err := s.watcher.Start(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	s.logger.Info("Sync service started. Press Ctrl+C to stop.")
	
	// Keep the service running until signal received
	select {
	case <-sigChan:
		// Received termination signal
	}

	s.logger.Info("Shutting down sync service...")
	return s.watcher.Stop()
}
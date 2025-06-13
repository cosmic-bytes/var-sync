package watcher

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"var-sync/internal/logger"
	"var-sync/internal/parser"
	"var-sync/pkg/models"
)

// FileWatcher provides thread-safe file watching with proper synchronization
// to prevent race conditions when multiple rules write to the same target file
type FileWatcher struct {
	watcher     *fsnotify.Watcher
	parser      *parser.Parser
	logger      *logger.Logger
	rules       []models.SyncRule
	debounce    time.Duration
	lastEvents  map[string]time.Time
	eventsMutex sync.RWMutex
	eventChan   chan models.SyncEvent
	stopChan    chan struct{}

	// Target file synchronization - prevents concurrent writes to same file
	targetFileMutexes map[string]*sync.Mutex
	targetMutex       sync.RWMutex

	// Batch processing for same-source-file changes
	batchProcessor *BatchProcessor
}

// BatchProcessor handles batching multiple rule changes from the same source file
type BatchProcessor struct {
	batches     map[string]*RuleBatch
	batchMutex  sync.Mutex
	batchDelay  time.Duration
	processChan chan string // Source file paths to process
}

// RuleBatch represents a batch of rules that need to be processed together
type RuleBatch struct {
	sourceFile string
	rules      []models.SyncRule
	timer      *time.Timer
	mutex      sync.Mutex
}

// New creates a new FileWatcher with proper synchronization
func New(logger *logger.Logger) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	fw := &FileWatcher{
		watcher:           watcher,
		parser:            parser.New(),
		logger:            logger,
		debounce:          500 * time.Millisecond,
		lastEvents:        make(map[string]time.Time),
		eventChan:         make(chan models.SyncEvent, 100),
		stopChan:          make(chan struct{}),
		targetFileMutexes: make(map[string]*sync.Mutex),
		batchProcessor: &BatchProcessor{
			batches:     make(map[string]*RuleBatch),
			batchDelay:  200 * time.Millisecond, // Batch rules for 200ms
			processChan: make(chan string, 100),
		},
	}

	return fw, nil
}

// getTargetFileMutex returns a mutex for the given target file, creating it if necessary
func (fw *FileWatcher) getTargetFileMutex(targetFile string) *sync.Mutex {
	absPath, err := filepath.Abs(targetFile)
	if err != nil {
		absPath = targetFile
	}

	fw.targetMutex.RLock()
	if mutex, exists := fw.targetFileMutexes[absPath]; exists {
		fw.targetMutex.RUnlock()
		return mutex
	}
	fw.targetMutex.RUnlock()

	fw.targetMutex.Lock()
	defer fw.targetMutex.Unlock()

	// Double-check pattern
	if mutex, exists := fw.targetFileMutexes[absPath]; exists {
		return mutex
	}

	mutex := &sync.Mutex{}
	fw.targetFileMutexes[absPath] = mutex
	return mutex
}

func (fw *FileWatcher) SetRules(rules []models.SyncRule) error {
	fw.eventsMutex.Lock()
	defer fw.eventsMutex.Unlock()

	fw.rules = rules

	watchedDirs := make(map[string]bool)
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		dir := filepath.Dir(rule.SourceFile)
		if !watchedDirs[dir] {
			if err := fw.watcher.Add(dir); err != nil {
				fw.logger.Error("Failed to watch directory: %s, error: %v", dir, err)
				continue
			}
			watchedDirs[dir] = true
			fw.logger.Info("Watching directory: %s for file: %s", dir, rule.SourceFile)
		}
	}

	return nil
}

func (fw *FileWatcher) Start() error {
	go fw.handleEvents()
	go fw.processEvents()
	go fw.processBatches()

	fw.logger.Info("Safe file watcher started")
	return nil
}

func (fw *FileWatcher) Stop() error {
	close(fw.stopChan)
	// Don't close eventChan as goroutines may still be writing to it
	// The consumer should drain the channel after stopping
	close(fw.batchProcessor.processChan)
	return fw.watcher.Close()
}

func (fw *FileWatcher) Events() <-chan models.SyncEvent {
	return fw.eventChan
}

func (fw *FileWatcher) handleEvents() {
	fw.logger.Debug("Starting safe event handler goroutine")
	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			fw.logger.Debug("Received file event: %s %s", event.Op, event.Name)
			if event.Op&fsnotify.Write == fsnotify.Write || 
			   event.Op&fsnotify.Create == fsnotify.Create || 
			   event.Op&fsnotify.Rename == fsnotify.Rename {
				fw.handleFileChange(event.Name)
			}

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			fw.logger.Error("File watcher error: %v", err)

		case <-fw.stopChan:
			return
		}
	}
}

func (fw *FileWatcher) handleFileChange(filename string) {
	fw.eventsMutex.RLock()
	defer fw.eventsMutex.RUnlock()

	now := time.Now()
	if lastEvent, exists := fw.lastEvents[filename]; exists {
		if now.Sub(lastEvent) < fw.debounce {
			return
		}
	}
	fw.lastEvents[filename] = now

	absPath, err := filepath.Abs(filename)
	if err != nil {
		fw.logger.Error("Failed to get absolute path for %s: %v", filename, err)
		return
	}

	// Find all rules that match this source file
	matchingRules := make([]models.SyncRule, 0)
	for _, rule := range fw.rules {
		if !rule.Enabled {
			continue
		}

		ruleAbsPath, err := filepath.Abs(rule.SourceFile)
		if err != nil {
			continue
		}

		if ruleAbsPath == absPath {
			matchingRules = append(matchingRules, rule)
		}
	}

	if len(matchingRules) > 0 {
		fw.logger.Debug("Found %d matching rules for file %s", len(matchingRules), filename)
		fw.batchRules(absPath, matchingRules)
	}
}

// batchRules groups rules by source file for batch processing
func (fw *FileWatcher) batchRules(sourceFile string, rules []models.SyncRule) {
	fw.batchProcessor.batchMutex.Lock()
	defer fw.batchProcessor.batchMutex.Unlock()

	batch, exists := fw.batchProcessor.batches[sourceFile]
	if !exists {
		batch = &RuleBatch{
			sourceFile: sourceFile,
			rules:      make([]models.SyncRule, 0),
		}
		fw.batchProcessor.batches[sourceFile] = batch
	}

	// Update rules in batch
	batch.mutex.Lock()
	batch.rules = rules
	
	// Reset or create timer
	if batch.timer != nil {
		batch.timer.Stop()
	}
	
	batch.timer = time.AfterFunc(fw.batchProcessor.batchDelay, func() {
		fw.batchProcessor.processChan <- sourceFile
	})
	batch.mutex.Unlock()

	fw.logger.Debug("Batched %d rules for source file %s", len(rules), sourceFile)
}

// processBatches handles batched rule processing
func (fw *FileWatcher) processBatches() {
	fw.logger.Debug("Starting batch processor goroutine")
	for {
		select {
		case sourceFile := <-fw.batchProcessor.processChan:
			fw.processBatch(sourceFile)
		case <-fw.stopChan:
			return
		}
	}
}

// processBatch processes all rules for a source file as a batch
func (fw *FileWatcher) processBatch(sourceFile string) {
	fw.batchProcessor.batchMutex.Lock()
	batch, exists := fw.batchProcessor.batches[sourceFile]
	if !exists {
		fw.batchProcessor.batchMutex.Unlock()
		return
	}
	delete(fw.batchProcessor.batches, sourceFile)
	fw.batchProcessor.batchMutex.Unlock()

	batch.mutex.Lock()
	rules := make([]models.SyncRule, len(batch.rules))
	copy(rules, batch.rules)
	batch.mutex.Unlock()

	fw.logger.Debug("Processing batch of %d rules for source file %s", len(rules), sourceFile)

	// Load source file once
	sourceData, err := fw.loadSourceFileWithRetry(sourceFile)
	if err != nil {
		fw.logger.Error("Failed to load source file %s: %v", sourceFile, err)
		for _, rule := range rules {
			fw.sendEvent(models.SyncEvent{
				RuleID:    rule.ID,
				Timestamp: time.Now(),
				Success:   false,
				Error:     fmt.Sprintf("Failed to load source file: %v", err),
			})
		}
		return
	}

	// Group rules by target file for synchronized writing
	targetGroups := make(map[string][]models.SyncRule)
	for _, rule := range rules {
		absTargetPath, err := filepath.Abs(rule.TargetFile)
		if err != nil {
			absTargetPath = rule.TargetFile
		}
		targetGroups[absTargetPath] = append(targetGroups[absTargetPath], rule)
	}

	// Process each target file group with proper synchronization
	for targetFile, targetRules := range targetGroups {
		fw.processTargetGroup(sourceData, targetFile, targetRules)
	}
}

// processTargetGroup processes all rules that write to the same target file
func (fw *FileWatcher) processTargetGroup(sourceData map[string]any, targetFile string, rules []models.SyncRule) {
	// Get mutex for this target file to ensure atomic operations
	targetMutex := fw.getTargetFileMutex(targetFile)
	targetMutex.Lock()
	defer targetMutex.Unlock()

	fw.logger.Debug("Processing %d rules for target file %s (synchronized)", len(rules), targetFile)

	// Collect all updates for batch surgical processing
	updates := make(map[string]any)
	allSuccessful := true
	events := make([]models.SyncEvent, 0, len(rules))

	for _, rule := range rules {
		event := fw.processRuleForBatch(sourceData, rule, updates)
		events = append(events, event)
		if !event.Success {
			allSuccessful = false
		}
	}

	// Apply all changes surgically to preserve formatting
	if allSuccessful && len(updates) > 0 {
		if err := fw.parser.UpdateFileValues(targetFile, updates); err != nil {
			fw.logger.Error("Failed to update target file %s: %v", targetFile, err)
			// Mark all events as failed
			for i := range events {
				events[i].Success = false
				events[i].Error = fmt.Sprintf("Failed to update target file: %v", err)
			}
		} else {
			fw.logger.Info("Successfully applied %d surgical updates to target file %s", len(updates), targetFile)
		}
	}

	// Send all events
	for _, event := range events {
		fw.sendEvent(event)
	}
}

// processRuleInBatch processes a single rule within a batch (without file I/O)
func (fw *FileWatcher) processRuleInBatch(sourceData, targetData map[string]any, rule models.SyncRule) models.SyncEvent {
	// Get source value
	newValue, err := fw.parser.GetValue(sourceData, rule.SourceKey)
	if err != nil {
		return models.SyncEvent{
			RuleID:    rule.ID,
			Timestamp: time.Now(),
			Success:   false,
			Error:     fmt.Sprintf("Failed to get source value: %v", err),
		}
	}

	// Get old value
	oldValue, _ := fw.parser.GetValue(targetData, rule.TargetKey)

	// Set new value
	if err := fw.parser.SetValue(targetData, rule.TargetKey, newValue); err != nil {
		return models.SyncEvent{
			RuleID:    rule.ID,
			Timestamp: time.Now(),
			Success:   false,
			Error:     fmt.Sprintf("Failed to set target value: %v", err),
		}
	}

	return models.SyncEvent{
		RuleID:    rule.ID,
		Timestamp: time.Now(),
		OldValue:  oldValue,
		NewValue:  newValue,
		Success:   true,
	}
}

// processRuleForBatch processes a single rule and collects updates for surgical batch processing
func (fw *FileWatcher) processRuleForBatch(sourceData map[string]any, rule models.SyncRule, updates map[string]any) models.SyncEvent {
	// Get source value
	newValue, err := fw.parser.GetValue(sourceData, rule.SourceKey)
	if err != nil {
		return models.SyncEvent{
			RuleID:    rule.ID,
			Timestamp: time.Now(),
			Success:   false,
			Error:     fmt.Sprintf("Failed to get source value: %v", err),
		}
	}

	// Get old value from the target file for the event
	var oldValue any
	if targetData, err := fw.parser.LoadFile(rule.TargetFile); err == nil {
		oldValue, _ = fw.parser.GetValue(targetData, rule.TargetKey)
	}

	// Add to updates map for surgical processing
	updates[rule.TargetKey] = newValue

	return models.SyncEvent{
		RuleID:    rule.ID,
		Timestamp: time.Now(),
		OldValue:  oldValue,
		NewValue:  newValue,
		Success:   true,
	}
}

// loadSourceFileWithRetry loads source file with retry logic
func (fw *FileWatcher) loadSourceFileWithRetry(sourceFile string) (map[string]any, error) {
	var sourceData map[string]any
	var err error
	
	for i := 0; i < 3; i++ {
		sourceData, err = fw.parser.LoadFile(sourceFile)
		if err == nil {
			return sourceData, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	
	return nil, err
}

func (fw *FileWatcher) processEvents() {
	fw.logger.Debug("Starting safe event processor goroutine")
	for {
		select {
		case event, ok := <-fw.eventChan:
			if !ok {
				return
			}
			
			if event.Success {
				fw.logger.Info("Safe sync successful for rule %s: %v -> %v", event.RuleID, event.OldValue, event.NewValue)
			} else {
				fw.logger.Error("Safe sync failed for rule %s: %s", event.RuleID, event.Error)
			}
		case <-fw.stopChan:
			return
		}
	}
}

func (fw *FileWatcher) sendEvent(event models.SyncEvent) {
	select {
	case fw.eventChan <- event:
	default:
		fw.logger.Warn("Event channel full, dropping event for rule: %s", event.RuleID)
	}
}
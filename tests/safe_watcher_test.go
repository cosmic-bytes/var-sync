// +build race

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"var-sync/internal/logger"
	"var-sync/internal/parser"
	"var-sync/internal/watcher"
	"var-sync/pkg/models"
)

// TestSafeWatcherRaceCondition tests that the SafeFileWatcher prevents race conditions
func TestSafeWatcherRaceCondition(t *testing.T) {
	tempDir := t.TempDir()
	
	sourceFile := filepath.Join(tempDir, "safe-source.yaml")
	targetFile := filepath.Join(tempDir, "safe-target.json")
	
	// Create source file with multiple values
	sourceContent := `database:
  host: localhost
  port: 5432
  username: admin
  password: secret
api:
  endpoint: http://localhost:8080
  timeout: 30
  retries: 3
  key: api-key-123`
	
	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	// Create target file
	targetContent := `{"config": {}}`
	if err := os.WriteFile(targetFile, []byte(targetContent), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}
	
	// Create multiple rules that all write to the same target file
	rules := []models.SyncRule{
		{
			ID:         "db-host-rule",
			Name:       "Database Host Rule",
			SourceFile: sourceFile,
			SourceKey:  "database.host",
			TargetFile: targetFile,
			TargetKey:  "config.db_host",
			Enabled:    true,
			Created:    time.Now(),
		},
		{
			ID:         "db-port-rule",
			Name:       "Database Port Rule", 
			SourceFile: sourceFile,
			SourceKey:  "database.port",
			TargetFile: targetFile,
			TargetKey:  "config.db_port",
			Enabled:    true,
			Created:    time.Now(),
		},
		{
			ID:         "db-username-rule",
			Name:       "Database Username Rule",
			SourceFile: sourceFile,
			SourceKey:  "database.username", 
			TargetFile: targetFile,
			TargetKey:  "config.db_username",
			Enabled:    true,
			Created:    time.Now(),
		},
		{
			ID:         "api-endpoint-rule",
			Name:       "API Endpoint Rule",
			SourceFile: sourceFile,
			SourceKey:  "api.endpoint",
			TargetFile: targetFile,
			TargetKey:  "config.api_endpoint",
			Enabled:    true,
			Created:    time.Now(),
		},
		{
			ID:         "api-timeout-rule",
			Name:       "API Timeout Rule",
			SourceFile: sourceFile,
			SourceKey:  "api.timeout",
			TargetFile: targetFile,
			TargetKey:  "config.api_timeout",
			Enabled:    true,
			Created:    time.Now(),
		},
	}
	
	// Set up safe watcher
	log := logger.New()
	log.SetLevel(logger.INFO)
	
	safeWatcher, err := watcher.New(log)
	if err != nil {
		t.Fatalf("Failed to create safe watcher: %v", err)
	}
	defer safeWatcher.Stop()
	
	if err := safeWatcher.SetRules(rules); err != nil {
		t.Fatalf("Failed to set rules: %v", err)
	}
	
	if err := safeWatcher.Start(); err != nil {
		t.Fatalf("Failed to start safe watcher: %v", err)
	}
	
	// Track sync events
	var mu sync.Mutex
	syncEvents := make([]models.SyncEvent, 0)
	
	// Monitor sync events
	go func() {
		for event := range safeWatcher.Events() {
			mu.Lock()
			syncEvents = append(syncEvents, event)
			mu.Unlock()
			
			t.Logf("Safe sync event: Rule=%s, Success=%v, OldValue=%v, NewValue=%v, Error=%s", 
				event.RuleID, event.Success, event.OldValue, event.NewValue, event.Error)
		}
	}()
	
	// Wait for initial setup
	time.Sleep(200 * time.Millisecond)
	
	// Simulate multiple rapid changes to the source file
	for i := 0; i < 3; i++ {
		updatedContent := fmt.Sprintf(`database:
  host: safe-host-%d
  port: %d
  username: user-%d
  password: secret-%d
api:
  endpoint: https://safe-api-%d.com
  timeout: %d
  retries: %d
  key: api-key-%d`, i, 5432+i, i, i, i, 30+i*10, 3+i, i)
		
		t.Logf("Writing update %d to source file...", i)
		if err := os.WriteFile(sourceFile, []byte(updatedContent), 0644); err != nil {
			t.Fatalf("Failed to update source file: %v", err)
		}
		
		// Small delay between updates
		time.Sleep(100 * time.Millisecond)
	}
	
	// Wait for all syncs to complete (including batching delay)
	time.Sleep(1 * time.Second)
	
	// Verify final target file state
	parser := parser.New()
	finalTargetData, err := parser.LoadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to load final target file: %v", err)
	}
	
	// Check that all expected values are present (should be from the last update)
	expectedValues := map[string]any{
		"config.db_host":      "safe-host-2",
		"config.db_port":      float64(5434), // JSON loads numbers as float64
		"config.db_username":  "user-2",
		"config.api_endpoint": "https://safe-api-2.com",
		"config.api_timeout":  float64(50),
	}
	
	mu.Lock()
	eventCount := len(syncEvents)
	mu.Unlock()
	
	t.Logf("Received %d sync events from safe watcher", eventCount)
	
	// Verify all values were synced correctly
	allCorrect := true
	for keyPath, expectedValue := range expectedValues {
		actualValue, err := parser.GetValue(finalTargetData, keyPath)
		if err != nil {
			t.Errorf("Failed to get value for %s: %v", keyPath, err)
			allCorrect = false
			continue
		}
		
		if actualValue != expectedValue {
			t.Errorf("Value mismatch for %s: expected %v, got %v", keyPath, expectedValue, actualValue)
			allCorrect = false
		} else {
			t.Logf("✓ Correctly synced %s = %v", keyPath, actualValue)
		}
	}
	
	// Check for any failed sync events
	mu.Lock()
	failedEvents := 0
	successfulEvents := 0
	for _, event := range syncEvents {
		if !event.Success {
			t.Errorf("Failed sync event: Rule=%s, Error=%s", event.RuleID, event.Error)
			failedEvents++
		} else {
			successfulEvents++
		}
	}
	mu.Unlock()
	
	if failedEvents > 0 {
		t.Errorf("Found %d failed sync events", failedEvents)
	}
	
	if allCorrect && failedEvents == 0 {
		t.Logf("✅ SUCCESS: Safe watcher prevented race conditions. All %d rules synced correctly with %d successful events.", 
			len(rules), successfulEvents)
	}
	
	t.Logf("Final target file contents: %+v", finalTargetData)
}

// TestSafeWatcherVsUnsafeComparison compares safe vs unsafe approaches
func TestSafeWatcherVsUnsafeComparison(t *testing.T) {
	tempDir := t.TempDir()
	
	// Test unsafe approach (current implementation)
	t.Run("UnsafeApproach", func(t *testing.T) {
		targetFile := filepath.Join(tempDir, "unsafe-target.json")
		targetContent := `{"results": {}}`
		if err := os.WriteFile(targetFile, []byte(targetContent), 0644); err != nil {
			t.Fatalf("Failed to create target file: %v", err)
		}
		
		parser := parser.New()
		
		// Simulate concurrent rule processing (unsafe)
		simulateUnsafeRuleSync := func(ruleID string, key, value string) error {
			// Load-Modify-Save without synchronization
			targetData, err := parser.LoadFile(targetFile)
			if err != nil {
				return fmt.Errorf("rule %s: failed to load target: %v", ruleID, err)
			}
			
			if err := parser.SetValue(targetData, key, value); err != nil {
				return fmt.Errorf("rule %s: failed to set value: %v", ruleID, err)
			}
			
			if err := parser.SaveFile(targetFile, targetData); err != nil {
				return fmt.Errorf("rule %s: failed to save target: %v", ruleID, err)
			}
			
			return nil
		}
		
		var wg sync.WaitGroup
		errors := make(chan error, 10)
		
		// Run 10 concurrent "rules"
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				key := fmt.Sprintf("results.key_%d", id)
				value := fmt.Sprintf("value_%d", id)
				if err := simulateUnsafeRuleSync(fmt.Sprintf("rule-%d", id), key, value); err != nil {
					errors <- err
				}
			}(i)
		}
		
		wg.Wait()
		close(errors)
		
		errorCount := 0
		for err := range errors {
			t.Logf("Unsafe error: %v", err)
			errorCount++
		}
		
		// Check final state
		finalData, err := parser.LoadFile(targetFile)
		if err != nil {
			t.Fatalf("Failed to load final unsafe target: %v", err)
		}
		
		// Count successful writes
		successfulWrites := 0
		if results, ok := finalData["results"].(map[string]any); ok {
			successfulWrites = len(results)
		}
		
		t.Logf("UNSAFE: %d successful writes out of 10, %d errors", successfulWrites, errorCount)
		
		if successfulWrites < 10 {
			t.Logf("UNSAFE APPROACH LOST DATA: Only %d/10 writes succeeded", successfulWrites)
		}
	})
	
	// Test safe approach
	t.Run("SafeApproach", func(t *testing.T) {
		targetFile := filepath.Join(tempDir, "safe-target.json")
		targetContent := `{"results": {}}`
		if err := os.WriteFile(targetFile, []byte(targetContent), 0644); err != nil {
			t.Fatalf("Failed to create target file: %v", err)
		}
		
		parser := parser.New()
		
		// Simulate safe batch processing
		mutex := &sync.Mutex{}
		
		simulateSafeRuleSync := func(ruleID string, key, value string) error {
			// Synchronized Load-Modify-Save
			mutex.Lock()
			defer mutex.Unlock()
			
			targetData, err := parser.LoadFile(targetFile)
			if err != nil {
				return fmt.Errorf("rule %s: failed to load target: %v", ruleID, err)
			}
			
			if err := parser.SetValue(targetData, key, value); err != nil {
				return fmt.Errorf("rule %s: failed to set value: %v", ruleID, err)
			}
			
			if err := parser.SaveFile(targetFile, targetData); err != nil {
				return fmt.Errorf("rule %s: failed to save target: %v", ruleID, err)
			}
			
			return nil
		}
		
		var wg sync.WaitGroup
		errors := make(chan error, 10)
		
		// Run 10 concurrent "rules" with synchronization
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				key := fmt.Sprintf("results.key_%d", id)
				value := fmt.Sprintf("value_%d", id)
				if err := simulateSafeRuleSync(fmt.Sprintf("rule-%d", id), key, value); err != nil {
					errors <- err
				}
			}(i)
		}
		
		wg.Wait()
		close(errors)
		
		errorCount := 0
		for err := range errors {
			t.Logf("Safe error: %v", err)
			errorCount++
		}
		
		// Check final state
		finalData, err := parser.LoadFile(targetFile)
		if err != nil {
			t.Fatalf("Failed to load final safe target: %v", err)
		}
		
		// Count successful writes
		successfulWrites := 0
		if results, ok := finalData["results"].(map[string]any); ok {
			successfulWrites = len(results)
		}
		
		t.Logf("SAFE: %d successful writes out of 10, %d errors", successfulWrites, errorCount)
		
		if successfulWrites == 10 && errorCount == 0 {
			t.Logf("✅ SAFE APPROACH SUCCEEDED: All 10/10 writes completed successfully")
		}
	})
}
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

// TestRaceConditionScenario tests the specific scenario where two variables 
// from the same source file write to the same target file simultaneously
func TestRaceConditionScenario(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create source and target files
	sourceFile := filepath.Join(tempDir, "source.yaml")
	targetFile := filepath.Join(tempDir, "target.json")
	
	// Initial source content
	sourceContent := `database:
  host: localhost
  port: 5432
api:
  endpoint: http://localhost:8080
  timeout: 30`
	
	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	// Initial target content
	targetContent := `{
  "config": {
    "db_host": "old-host",
    "db_port": 3306,
    "api_endpoint": "old-endpoint",
    "api_timeout": 10
  }
}`
	
	if err := os.WriteFile(targetFile, []byte(targetContent), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}
	
	// Create multiple rules that sync different keys from same source to same target
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
	
	// Set up logger and watcher
	log := logger.New()
	log.SetLevel(logger.DEBUG)
	
	fw, err := watcher.New(log)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer fw.Stop()
	
	if err := fw.SetRules(rules); err != nil {
		t.Fatalf("Failed to set rules: %v", err)
	}
	
	if err := fw.Start(); err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}
	
	// Wait for initial setup
	time.Sleep(100 * time.Millisecond)
	
	// Track sync events
	var mu sync.Mutex
	syncEvents := make([]models.SyncEvent, 0)
	
	// Monitor sync events
	go func() {
		for event := range fw.Events() {
			mu.Lock()
			syncEvents = append(syncEvents, event)
			mu.Unlock()
			
			t.Logf("Sync event: Rule=%s, Success=%v, OldValue=%v, NewValue=%v, Error=%s", 
				event.RuleID, event.Success, event.OldValue, event.NewValue, event.Error)
		}
	}()
	
	// Simulate a file change that should trigger all rules
	updatedContent := `database:
  host: production-db.example.com
  port: 5433
api:
  endpoint: https://api.production.com
  timeout: 60`
	
	t.Log("Writing updated content to trigger multiple sync rules...")
	if err := os.WriteFile(sourceFile, []byte(updatedContent), 0644); err != nil {
		t.Fatalf("Failed to update source file: %v", err)
	}
	
	// Wait for syncs to complete
	time.Sleep(2 * time.Second)
	
	// Verify final target file state
	parser := parser.New()
	finalTargetData, err := parser.LoadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to load final target file: %v", err)
	}
	
	// Check that all expected values are present
	expectedValues := map[string]any{
		"config.db_host":      "production-db.example.com",
		"config.db_port":      float64(5433), // JSON loads numbers as float64
		"config.api_endpoint": "https://api.production.com", 
		"config.api_timeout":  float64(60),
	}
	
	mu.Lock()
	eventCount := len(syncEvents)
	mu.Unlock()
	
	t.Logf("Received %d sync events", eventCount)
	
	// Verify all values were synced correctly
	for keyPath, expectedValue := range expectedValues {
		actualValue, err := parser.GetValue(finalTargetData, keyPath)
		if err != nil {
			t.Errorf("Failed to get value for %s: %v", keyPath, err)
			continue
		}
		
		if actualValue != expectedValue {
			t.Errorf("Value mismatch for %s: expected %v, got %v", keyPath, expectedValue, actualValue)
		} else {
			t.Logf("✓ Correctly synced %s = %v", keyPath, actualValue)
		}
	}
	
	// Check for any failed sync events
	mu.Lock()
	failedEvents := 0
	for _, event := range syncEvents {
		if !event.Success {
			t.Errorf("Failed sync event: Rule=%s, Error=%s", event.RuleID, event.Error)
			failedEvents++
		}
	}
	mu.Unlock()
	
	if failedEvents > 0 {
		t.Errorf("Found %d failed sync events", failedEvents)
	}
	
	t.Logf("Race condition test completed. All %d rules should have synced successfully.", len(rules))
}

// TestConcurrentTargetFileWrites tests the race condition more directly
func TestConcurrentTargetFileWrites(t *testing.T) {
	tempDir := t.TempDir()
	
	targetFile := filepath.Join(tempDir, "concurrent-target.json")
	
	// Create target file
	targetContent := `{"results": {}}`
	if err := os.WriteFile(targetFile, []byte(targetContent), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}
	
	parser := parser.New()
	
	// Simulate what happens when multiple rules try to sync to same target
	// This simulates the processRule function's load-modify-save pattern
	simulateRuleSync := func(ruleID string, key, value string) error {
		// Load target file (like processRule does)
		targetData, err := parser.LoadFile(targetFile)
		if err != nil {
			return fmt.Errorf("rule %s: failed to load target: %v", ruleID, err)
		}
		
		// Set value (like processRule does)
		if err := parser.SetValue(targetData, key, value); err != nil {
			return fmt.Errorf("rule %s: failed to set value: %v", ruleID, err)
		}
		
		// Save file (like processRule does)
		if err := parser.SaveFile(targetFile, targetData); err != nil {
			return fmt.Errorf("rule %s: failed to save target: %v", ruleID, err)
		}
		
		return nil
	}
	
	// Run multiple "rules" concurrently that write to same target
	var wg sync.WaitGroup
	errors := make(chan error, 5)
	
	rules := []struct{ key, value string }{
		{"results.key1", "value1"},
		{"results.key2", "value2"}, 
		{"results.key3", "value3"},
		{"results.key4", "value4"},
		{"results.key5", "value5"},
	}
	
	start := time.Now()
	
	for i, rule := range rules {
		wg.Add(1)
		go func(ruleID string, key, value string) {
			defer wg.Done()
			if err := simulateRuleSync(ruleID, key, value); err != nil {
				errors <- err
			}
		}(fmt.Sprintf("rule-%d", i+1), rule.key, rule.value)
	}
	
	wg.Wait()
	close(errors)
	
	duration := time.Since(start)
	t.Logf("Concurrent processing took %v", duration)
	
	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Logf("Sync error: %v", err)
		errorCount++
	}
	
	// Verify final state
	finalData, err := parser.LoadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to load final target file: %v", err)
	}
	
	// Check that all 5 values were written
	missingKeys := 0
	for i := 1; i <= 5; i++ {
		keyPath := fmt.Sprintf("results.key%d", i)
		expectedValue := fmt.Sprintf("value%d", i)
		
		actualValue, err := parser.GetValue(finalData, keyPath)
		if err != nil {
			t.Logf("Missing key %s: %v", keyPath, err)
			missingKeys++
			continue
		}
		
		if actualValue != expectedValue {
			t.Errorf("Wrong value for %s: expected %s, got %v", keyPath, expectedValue, actualValue)
		} else {
			t.Logf("✓ Successfully wrote %s = %v", keyPath, actualValue)
		}
	}
	
	t.Logf("Final target file contents: %+v", finalData)
	
	if missingKeys > 0 {
		t.Errorf("RACE CONDITION DETECTED: %d keys were lost due to concurrent writes", missingKeys)
	}
	
	if errorCount > 0 {
		t.Logf("Found %d errors during concurrent processing", errorCount)
	}
}

// TestFileCorruptionDetection tests if we can detect file corruption during concurrent writes
func TestFileCorruptionDetection(t *testing.T) {
	tempDir := t.TempDir()
	
	targetFile := filepath.Join(tempDir, "corruption-test.json")
	
	// Create initial target file
	initialContent := `{"data": {}}`
	if err := os.WriteFile(targetFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}
	
	parser := parser.New()
	
	// Simulate many concurrent writes to the same file
	const numGoroutines = 20
	const writesPerGoroutine = 10
	
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*writesPerGoroutine)
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < writesPerGoroutine; j++ {
				// Load file
				data, err := parser.LoadFile(targetFile)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, write %d: load failed: %v", goroutineID, j, err)
					continue
				}
				
				// Set a unique value
				key := fmt.Sprintf("data.g%d_w%d", goroutineID, j)
				value := fmt.Sprintf("value_%d_%d_%d", goroutineID, j, time.Now().UnixNano())
				
				if err := parser.SetValue(data, key, value); err != nil {
					errors <- fmt.Errorf("goroutine %d, write %d: set failed: %v", goroutineID, j, err)
					continue
				}
				
				// Save file
				if err := parser.SaveFile(targetFile, data); err != nil {
					errors <- fmt.Errorf("goroutine %d, write %d: save failed: %v", goroutineID, j, err)
					continue
				}
				
				// Small delay to increase chance of collision
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}
	
	wg.Wait()
	close(errors)
	
	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Logf("Concurrent write error: %v", err)
		errorCount++
	}
	
	if errorCount > 0 {
		t.Logf("Found %d errors during concurrent writes", errorCount)
	}
	
	// Try to load the final file to see if it's corrupted
	finalData, err := parser.LoadFile(targetFile)
	if err != nil {
		t.Errorf("Final file is corrupted and cannot be loaded: %v", err)
	} else {
		t.Logf("Final file loaded successfully with %d top-level keys", len(finalData))
		
		// Count how many values were actually written
		if dataMap, ok := finalData["data"].(map[string]any); ok {
			t.Logf("Successfully wrote %d values to target file", len(dataMap))
		}
	}
}
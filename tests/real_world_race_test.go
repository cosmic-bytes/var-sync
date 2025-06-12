package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"var-sync/internal/logger"
	"var-sync/internal/parser"
	"var-sync/internal/watcher"
	"var-sync/pkg/models"
)

// TestRealWorldRaceConditionFixed tests that the file watcher prevents race conditions
// when multiple rules respond to the same file change event
func TestRealWorldRaceConditionFixed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real-world race condition test in short mode")
	}

	tempDir := t.TempDir()
	
	// Create source and target files
	sourceFile := filepath.Join(tempDir, "config.yaml")
	targetFile := filepath.Join(tempDir, "app.json")
	
	// Initial source content
	sourceContent := `database:
  host: localhost
  port: 5432
  username: admin
  password: secret
api:
  endpoint: http://localhost:8080
  timeout: 30
  retries: 3
cache:
  redis_host: localhost
  redis_port: 6379
  ttl: 3600`
	
	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	// Initial target content
	targetContent := `{
  "config": {
    "db_host": "old-host",
    "db_port": 3306,
    "db_user": "old-user",
    "api_endpoint": "old-endpoint",
    "api_timeout": 10,
    "api_retries": 1,
    "cache_host": "old-cache",
    "cache_port": 6380,
    "cache_ttl": 1800
  }
}`
	
	if err := os.WriteFile(targetFile, []byte(targetContent), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}
	
	// Create multiple rules that all sync to the same target file
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
			TargetKey:  "config.db_user",
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
		{
			ID:         "api-retries-rule",
			Name:       "API Retries Rule",
			SourceFile: sourceFile,
			SourceKey:  "api.retries",
			TargetFile: targetFile,
			TargetKey:  "config.api_retries",
			Enabled:    true,
			Created:    time.Now(),
		},
		{
			ID:         "cache-host-rule",
			Name:       "Cache Host Rule",
			SourceFile: sourceFile,
			SourceKey:  "cache.redis_host",
			TargetFile: targetFile,
			TargetKey:  "config.cache_host",
			Enabled:    true,
			Created:    time.Now(),
		},
		{
			ID:         "cache-port-rule",
			Name:       "Cache Port Rule",
			SourceFile: sourceFile,
			SourceKey:  "cache.redis_port",
			TargetFile: targetFile,
			TargetKey:  "config.cache_port",
			Enabled:    true,
			Created:    time.Now(),
		},
		{
			ID:         "cache-ttl-rule",
			Name:       "Cache TTL Rule",
			SourceFile: sourceFile,
			SourceKey:  "cache.ttl",
			TargetFile: targetFile,
			TargetKey:  "config.cache_ttl",
			Enabled:    true,
			Created:    time.Now(),
		},
	}
	
	// Set up file watcher (now using the safe implementation)
	log := logger.New()
	log.SetLevel(logger.INFO)
	
	fw, err := watcher.New(log)
	if err != nil {
		t.Fatalf("Failed to create file watcher: %v", err)
	}
	defer fw.Stop()
	
	if err := fw.SetRules(rules); err != nil {
		t.Fatalf("Failed to set rules: %v", err)
	}
	
	if err := fw.Start(); err != nil {
		t.Fatalf("Failed to start file watcher: %v", err)
	}
	
	// Monitor sync events
	successfulSyncs := 0
	failedSyncs := 0
	
	go func() {
		for event := range fw.Events() {
			if event.Success {
				successfulSyncs++
				t.Logf("✓ Successful sync: %s -> %v", event.RuleID, event.NewValue)
			} else {
				failedSyncs++
				t.Logf("✗ Failed sync: %s - %s", event.RuleID, event.Error)
			}
		}
	}()
	
	// Wait for initial setup
	time.Sleep(300 * time.Millisecond)
	
	// Trigger file change that should activate all 9 rules simultaneously
	t.Log("Triggering file change that should sync all 9 rules...")
	
	updatedContent := `database:
  host: production-db.example.com
  port: 5433
  username: prod_user
  password: prod_secret
api:
  endpoint: https://api.production.com
  timeout: 60
  retries: 5
cache:
  redis_host: redis.production.com
  redis_port: 6379
  ttl: 7200`
	
	if err := os.WriteFile(sourceFile, []byte(updatedContent), 0644); err != nil {
		t.Fatalf("Failed to update source file: %v", err)
	}
	
	// Wait for all syncs to complete (including batch processing delay)
	time.Sleep(1 * time.Second)
	
	// Verify final state
	parser := parser.New()
	finalData, err := parser.LoadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to load final target file: %v", err)
	}
	
	// Expected values (after sync)
	expectedValues := map[string]any{
		"config.db_host":      "production-db.example.com",
		"config.db_port":      float64(5433), // JSON loads numbers as float64
		"config.db_user":      "prod_user",
		"config.api_endpoint": "https://api.production.com",
		"config.api_timeout":  float64(60),
		"config.api_retries":  float64(5),
		"config.cache_host":   "redis.production.com",
		"config.cache_port":   float64(6379),
		"config.cache_ttl":    float64(7200),
	}
	
	// Verify all values were synced correctly
	correctSyncs := 0
	incorrectSyncs := 0
	
	for keyPath, expectedValue := range expectedValues {
		actualValue, err := parser.GetValue(finalData, keyPath)
		if err != nil {
			t.Errorf("MISSING: %s (error: %v)", keyPath, err)
			incorrectSyncs++
			continue
		}
		
		if actualValue != expectedValue {
			t.Errorf("WRONG VALUE: %s expected %v, got %v", keyPath, expectedValue, actualValue)
			incorrectSyncs++
		} else {
			t.Logf("✓ CORRECT: %s = %v", keyPath, actualValue)
			correctSyncs++
		}
	}
	
	t.Logf("\n=== FINAL RESULTS ===")
	t.Logf("Rules configured: %d", len(rules))
	t.Logf("Successful sync events: %d", successfulSyncs)
	t.Logf("Failed sync events: %d", failedSyncs)
	t.Logf("Correct final values: %d", correctSyncs)
	t.Logf("Incorrect final values: %d", incorrectSyncs)
	
	// Test assertions
	if correctSyncs == len(expectedValues) && incorrectSyncs == 0 && failedSyncs == 0 {
		t.Logf("🎉 SUCCESS: All %d rules synced correctly with NO race conditions!", len(rules))
	} else {
		t.Errorf("❌ FAILURE: Race condition protection failed")
		t.Errorf("   Expected all %d values to sync correctly", len(expectedValues))
		t.Errorf("   Got %d correct, %d incorrect, %d failed", correctSyncs, incorrectSyncs, failedSyncs)
	}
	
	// Additional verification: Check that no data was lost
	if configSection, ok := finalData["config"].(map[string]any); ok {
		if len(configSection) < len(expectedValues) {
			t.Errorf("Data loss detected: only %d values in target, expected %d", 
				len(configSection), len(expectedValues))
		}
	}
}
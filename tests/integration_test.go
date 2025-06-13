package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"var-sync/internal/config"
	"var-sync/internal/logger"
	"var-sync/internal/parser"
	"var-sync/pkg/models"
)

// TestIntegrationFullSync tests the complete sync flow
func TestIntegrationFullSync(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	sourceFile := filepath.Join(tempDir, "source.yaml")
	targetFile := filepath.Join(tempDir, "target.json")
	configFile := filepath.Join(tempDir, "config.json")

	// Create source YAML file
	sourceContent := `database:
  host: localhost
  port: 5432
api:
  key: test-key-123
  timeout: 30`

	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create target JSON file
	targetContent := `{
  "config": {
    "db": {
      "host": "old-host",
      "port": 3306
    },
    "api": {
      "secret": "old-secret"
    }
  }
}`

	if err := os.WriteFile(targetFile, []byte(targetContent), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create sync rule
	rule := models.SyncRule{
		ID:          "test-sync-rule",
		Name:        "Test Database Host Sync",
		Description: "Sync database host from YAML to JSON",
		SourceFile:  sourceFile,
		SourceKey:   "database.host",
		TargetFile:  targetFile,
		TargetKey:   "config.db.host",
		Enabled:     true,
		Created:     time.Now(),
	}

	// Create config
	cfg := &models.Config{
		Rules:   []models.SyncRule{rule},
		LogFile: filepath.Join(tempDir, "test.log"),
		Debug:   false,
	}

	// Save config
	if err := config.Save(cfg, configFile); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Test the sync process
	parser := parser.New()
	logger := logger.New()

	// Load source data
	sourceData, err := parser.LoadFile(sourceFile)
	if err != nil {
		t.Fatalf("Failed to load source file: %v", err)
	}

	// Get source value
	sourceValue, err := parser.GetValue(sourceData, rule.SourceKey)
	if err != nil {
		t.Fatalf("Failed to get source value: %v", err)
	}

	// Load target data
	targetData, err := parser.LoadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to load target file: %v", err)
	}

	// Set target value
	if err := parser.SetValue(targetData, rule.TargetKey, sourceValue); err != nil {
		t.Fatalf("Failed to set target value: %v", err)
	}

	// Save target file
	if err := parser.SaveFile(targetFile, targetData); err != nil {
		t.Fatalf("Failed to save target file: %v", err)
	}

	// Verify the sync worked
	updatedTargetData, err := parser.LoadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to load updated target file: %v", err)
	}

	actualValue, err := parser.GetValue(updatedTargetData, rule.TargetKey)
	if err != nil {
		t.Fatalf("Failed to get updated target value: %v", err)
	}

	if actualValue != "localhost" {
		t.Errorf("Expected target value 'localhost', got %v", actualValue)
	}

	logger.Info("Integration test completed successfully")
}

// TestIntegrationMultipleRules tests syncing with multiple rules
func TestIntegrationMultipleRules(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	sourceFile := filepath.Join(tempDir, "app-config.toml")
	targetFile1 := filepath.Join(tempDir, "target1.json")
	targetFile2 := filepath.Join(tempDir, "target2.yaml")

	// Create source TOML file
	sourceContent := `[database]
host = "db.example.com"
port = 5432
username = "admin"

[redis]
host = "redis.example.com"
port = 6379`

	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create target files
	target1Content := `{"db": {"host": "old-host"}}`
	if err := os.WriteFile(targetFile1, []byte(target1Content), 0644); err != nil {
		t.Fatalf("Failed to create target file 1: %v", err)
	}

	target2Content := `cache:
  host: old-cache-host
  port: 6380`
	if err := os.WriteFile(targetFile2, []byte(target2Content), 0644); err != nil {
		t.Fatalf("Failed to create target file 2: %v", err)
	}

	// Create sync rules
	rules := []models.SyncRule{
		{
			ID:         "db-host-sync",
			Name:       "Database Host Sync",
			SourceFile: sourceFile,
			SourceKey:  "database.host",
			TargetFile: targetFile1,
			TargetKey:  "db.host",
			Enabled:    true,
			Created:    time.Now(),
		},
		{
			ID:         "redis-host-sync",
			Name:       "Redis Host Sync",
			SourceFile: sourceFile,
			SourceKey:  "redis.host",
			TargetFile: targetFile2,
			TargetKey:  "cache.host",
			Enabled:    true,
			Created:    time.Now(),
		},
	}

	parser := parser.New()

	// Perform sync for each rule
	for _, rule := range rules {
		// Load source data
		sourceData, err := parser.LoadFile(rule.SourceFile)
		if err != nil {
			t.Fatalf("Failed to load source file for rule %s: %v", rule.ID, err)
		}

		// Get source value
		sourceValue, err := parser.GetValue(sourceData, rule.SourceKey)
		if err != nil {
			t.Fatalf("Failed to get source value for rule %s: %v", rule.ID, err)
		}

		// Load target data
		targetData, err := parser.LoadFile(rule.TargetFile)
		if err != nil {
			t.Fatalf("Failed to load target file for rule %s: %v", rule.ID, err)
		}

		// Set target value
		if err := parser.SetValue(targetData, rule.TargetKey, sourceValue); err != nil {
			t.Fatalf("Failed to set target value for rule %s: %v", rule.ID, err)
		}

		// Save target file
		if err := parser.SaveFile(rule.TargetFile, targetData); err != nil {
			t.Fatalf("Failed to save target file for rule %s: %v", rule.ID, err)
		}
	}

	// Verify first sync (TOML -> JSON)
	target1Data, err := parser.LoadFile(targetFile1)
	if err != nil {
		t.Fatalf("Failed to load updated target file 1: %v", err)
	}

	dbHost, err := parser.GetValue(target1Data, "db.host")
	if err != nil {
		t.Fatalf("Failed to get db.host from target file 1: %v", err)
	}

	if dbHost != "db.example.com" {
		t.Errorf("Expected db.host 'db.example.com', got %v", dbHost)
	}

	// Verify second sync (TOML -> YAML)
	target2Data, err := parser.LoadFile(targetFile2)
	if err != nil {
		t.Fatalf("Failed to load updated target file 2: %v", err)
	}

	cacheHost, err := parser.GetValue(target2Data, "cache.host")
	if err != nil {
		t.Fatalf("Failed to get cache.host from target file 2: %v", err)
	}

	if cacheHost != "redis.example.com" {
		t.Errorf("Expected cache.host 'redis.example.com', got %v", cacheHost)
	}
}

// TestIntegrationConfigManager tests the config manager integration
func TestIntegrationConfigManager(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "integration-config.json")

	// Create a config manager
	manager, err := config.NewManager(configFile)
	if err != nil {
		t.Fatalf("Failed to create config manager: %v", err)
	}

	// Add a rule
	rule := models.SyncRule{
		ID:          "integration-rule",
		Name:        "Integration Test Rule",
		Description: "Test rule for integration testing",
		SourceFile:  "source.yaml",
		SourceKey:   "app.version",
		TargetFile:  "target.json",
		TargetKey:   "version",
		Enabled:     true,
		Created:     time.Now(),
	}

	manager.AddRule(rule)

	// Save the config
	if err := manager.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Create a new manager to test loading
	manager2, err := config.NewManager(configFile)
	if err != nil {
		t.Fatalf("Failed to create second config manager: %v", err)
	}

	// Verify the rule was loaded
	loadedRule := manager2.GetRule("integration-rule")
	if loadedRule == nil {
		t.Fatal("Rule was not loaded by second manager")
	}

	if loadedRule.Name != "Integration Test Rule" {
		t.Errorf("Expected rule name 'Integration Test Rule', got %s", loadedRule.Name)
	}

	// Test removing the rule
	manager2.RemoveRule("integration-rule")

	if len(manager2.Config().Rules) != 0 {
		t.Errorf("Expected 0 rules after removal, got %d", len(manager2.Config().Rules))
	}
}

// TestIntegrationErrorHandling tests error scenarios
func TestIntegrationErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	parser := parser.New()

	// Test loading non-existent file
	_, err := parser.LoadFile(filepath.Join(tempDir, "nonexistent.json"))
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}

	// Test invalid JSON file
	invalidJSONFile := filepath.Join(tempDir, "invalid.json")
	if err := os.WriteFile(invalidJSONFile, []byte("invalid json content"), 0644); err != nil {
		t.Fatalf("Failed to create invalid JSON file: %v", err)
	}

	_, err = parser.LoadFile(invalidJSONFile)
	if err == nil {
		t.Error("Expected error when loading invalid JSON file")
	}

	// Test invalid YAML file
	invalidYAMLFile := filepath.Join(tempDir, "invalid.yaml")
	if err := os.WriteFile(invalidYAMLFile, []byte("invalid:\n  yaml:\ncontent"), 0644); err != nil {
		t.Fatalf("Failed to create invalid YAML file: %v", err)
	}

	_, err = parser.LoadFile(invalidYAMLFile)
	if err == nil {
		t.Error("Expected error when loading invalid YAML file")
	}

	// Test getting non-existent key
	validData := map[string]any{
		"existing": "value",
	}

	_, err = parser.GetValue(validData, "nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent key")
	}

	// Test setting value on conflicting path
	err = parser.SetValue(validData, "existing.nested", "value")
	if err == nil {
		t.Error("Expected error when setting nested value on string")
	}
}

// TestIntegrationRealWorldScenario simulates a real-world usage scenario
func TestIntegrationRealWorldScenario(t *testing.T) {
	tempDir := t.TempDir()

	// Simulate a microservice configuration scenario
	// - Main config file (YAML) contains database and API settings
	// - Service-specific config files (JSON) need to sync certain values
	// - Docker compose file (TOML) needs to sync port numbers

	mainConfigFile := filepath.Join(tempDir, "main-config.yaml")
	serviceConfigFile := filepath.Join(tempDir, "service-config.json")
	dockerConfigFile := filepath.Join(tempDir, "docker-compose.toml")

	// Create main config
	mainConfigContent := `database:
  host: production-db.example.com
  port: 5432
  username: service_user
  ssl: true

api:
  base_url: https://api.example.com
  version: v2
  timeout: 30
  rate_limit: 1000

monitoring:
  enabled: true
  endpoint: https://monitoring.example.com
  interval: 60`

	if err := os.WriteFile(mainConfigFile, []byte(mainConfigContent), 0644); err != nil {
		t.Fatalf("Failed to create main config file: %v", err)
	}

	// Create service config
	serviceConfigContent := `{
  "service": {
    "name": "user-service",
    "version": "1.0.0"
  },
  "database": {
    "host": "localhost",
    "port": 3306,
    "ssl": false
  },
  "api": {
    "base_url": "http://localhost:8080",
    "timeout": 10
  },
  "monitoring": {
    "enabled": false
  }
}`

	if err := os.WriteFile(serviceConfigFile, []byte(serviceConfigContent), 0644); err != nil {
		t.Fatalf("Failed to create service config file: %v", err)
	}

	// Create docker config
	dockerConfigContent := `[database]
host = "localhost"
port = 3306

[api]
port = 8080

[monitoring]
enabled = false`

	if err := os.WriteFile(dockerConfigFile, []byte(dockerConfigContent), 0644); err != nil {
		t.Fatalf("Failed to create docker config file: %v", err)
	}

	// Define sync rules
	rules := []models.SyncRule{
		{
			ID:         "db-host-to-service",
			Name:       "Database Host to Service",
			SourceFile: mainConfigFile,
			SourceKey:  "database.host",
			TargetFile: serviceConfigFile,
			TargetKey:  "database.host",
			Enabled:    true,
			Created:    time.Now(),
		},
		{
			ID:         "db-ssl-to-service",
			Name:       "Database SSL to Service",
			SourceFile: mainConfigFile,
			SourceKey:  "database.ssl",
			TargetFile: serviceConfigFile,
			TargetKey:  "database.ssl",
			Enabled:    true,
			Created:    time.Now(),
		},
		{
			ID:         "api-timeout-to-service",
			Name:       "API Timeout to Service",
			SourceFile: mainConfigFile,
			SourceKey:  "api.timeout",
			TargetFile: serviceConfigFile,
			TargetKey:  "api.timeout",
			Enabled:    true,
			Created:    time.Now(),
		},
		{
			ID:         "monitoring-enabled-to-docker",
			Name:       "Monitoring Enabled to Docker",
			SourceFile: mainConfigFile,
			SourceKey:  "monitoring.enabled",
			TargetFile: dockerConfigFile,
			TargetKey:  "monitoring.enabled",
			Enabled:    true,
			Created:    time.Now(),
		},
	}

	parser := parser.New()
	log := logger.New()
	log.SetLevel(logger.DEBUG)

	// Perform all syncs
	for _, rule := range rules {
		log.Info("Processing rule: %s", rule.Name)

		// Load source data
		sourceData, err := parser.LoadFile(rule.SourceFile)
		if err != nil {
			t.Fatalf("Failed to load source file for rule %s: %v", rule.ID, err)
		}

		// Get source value
		sourceValue, err := parser.GetValue(sourceData, rule.SourceKey)
		if err != nil {
			t.Fatalf("Failed to get source value for rule %s: %v", rule.ID, err)
		}

		log.Debug("Source value for %s: %v", rule.SourceKey, sourceValue)

		// Load target data
		targetData, err := parser.LoadFile(rule.TargetFile)
		if err != nil {
			t.Fatalf("Failed to load target file for rule %s: %v", rule.ID, err)
		}

		// Set target value
		if err := parser.SetValue(targetData, rule.TargetKey, sourceValue); err != nil {
			t.Fatalf("Failed to set target value for rule %s: %v", rule.ID, err)
		}

		// Save target file
		if err := parser.SaveFile(rule.TargetFile, targetData); err != nil {
			t.Fatalf("Failed to save target file for rule %s: %v", rule.ID, err)
		}

		log.Info("Successfully synced %s -> %s", rule.SourceKey, rule.TargetKey)
	}

	// Verify all syncs worked correctly
	verifications := []struct {
		file     string
		key      string
		expected any
	}{
		{serviceConfigFile, "database.host", "production-db.example.com"},
		{serviceConfigFile, "database.ssl", true},
		{serviceConfigFile, "api.timeout", float64(30)}, // JSON loads numbers as float64
		{dockerConfigFile, "monitoring.enabled", true},
	}

	for _, v := range verifications {
		data, err := parser.LoadFile(v.file)
		if err != nil {
			t.Fatalf("Failed to load file %s for verification: %v", v.file, err)
		}

		actual, err := parser.GetValue(data, v.key)
		if err != nil {
			t.Fatalf("Failed to get value %s from %s: %v", v.key, v.file, err)
		}

		if actual != v.expected {
			t.Errorf("Verification failed for %s:%s - expected %v, got %v",
				filepath.Base(v.file), v.key, v.expected, actual)
		}
	}

	log.Info("All verifications passed - real-world scenario test completed")
}

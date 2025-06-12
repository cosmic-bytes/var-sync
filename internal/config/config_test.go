package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"var-sync/pkg/models"
)

func TestNew(t *testing.T) {
	cfg := New()
	
	if cfg == nil {
		t.Fatal("New() returned nil")
	}
	
	if cfg.Rules == nil {
		t.Error("Rules slice is nil")
	}
	
	if len(cfg.Rules) != 0 {
		t.Errorf("Expected empty rules slice, got %d rules", len(cfg.Rules))
	}
	
	if cfg.LogFile != "var-sync.log" {
		t.Errorf("Expected LogFile to be 'var-sync.log', got %s", cfg.LogFile)
	}
	
	if cfg.Debug {
		t.Error("Expected Debug to be false")
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.json")
	
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
	
	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

func TestLoadExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.json")
	
	// Create test config
	testRule := models.SyncRule{
		ID:          "test-rule",
		Name:        "Test Rule",
		Description: "Test description",
		SourceFile:  "source.yaml",
		SourceKey:   "test.key",
		TargetFile:  "target.json",
		TargetKey:   "test.target",
		Enabled:     true,
		Created:     time.Now(),
	}
	
	testConfig := &models.Config{
		Rules:   []models.SyncRule{testRule},
		LogFile: "test.log",
		Debug:   true,
	}
	
	// Save test config
	if err := Save(testConfig, configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}
	
	// Load config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	
	if len(cfg.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(cfg.Rules))
	}
	
	if cfg.Rules[0].ID != "test-rule" {
		t.Errorf("Expected rule ID 'test-rule', got %s", cfg.Rules[0].ID)
	}
	
	if cfg.LogFile != "test.log" {
		t.Errorf("Expected LogFile 'test.log', got %s", cfg.LogFile)
	}
	
	if !cfg.Debug {
		t.Error("Expected Debug to be true")
	}
}

func TestSave(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "subdir", "test-config.json")
	
	cfg := New()
	cfg.LogFile = "custom.log"
	cfg.Debug = true
	
	if err := Save(cfg, configPath); err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}
	
	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
	
	// Load and verify content
	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}
	
	if loadedCfg.LogFile != "custom.log" {
		t.Errorf("Expected LogFile 'custom.log', got %s", loadedCfg.LogFile)
	}
	
	if !loadedCfg.Debug {
		t.Error("Expected Debug to be true")
	}
}

func TestManager(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "manager-test.json")
	
	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("NewManager() returned error: %v", err)
	}
	
	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}
	
	// Test Config() method
	cfg := manager.Config()
	if cfg == nil {
		t.Error("Config() returned nil")
	}
	
	// Test AddRule
	testRule := models.SyncRule{
		ID:          "manager-test-rule",
		Name:        "Manager Test Rule",
		Description: "Test description",
		SourceFile:  "source.yaml",
		SourceKey:   "test.key",
		TargetFile:  "target.json",
		TargetKey:   "test.target",
		Enabled:     true,
		Created:     time.Now(),
	}
	
	manager.AddRule(testRule)
	
	if len(manager.Config().Rules) != 1 {
		t.Errorf("Expected 1 rule after AddRule, got %d", len(manager.Config().Rules))
	}
	
	// Test GetRule
	rule := manager.GetRule("manager-test-rule")
	if rule == nil {
		t.Error("GetRule() returned nil for existing rule")
	}
	
	if rule.ID != "manager-test-rule" {
		t.Errorf("Expected rule ID 'manager-test-rule', got %s", rule.ID)
	}
	
	// Test GetRule with non-existent ID
	nonExistentRule := manager.GetRule("non-existent")
	if nonExistentRule != nil {
		t.Error("GetRule() should return nil for non-existent rule")
	}
	
	// Test Save
	if err := manager.Save(); err != nil {
		t.Errorf("Save() returned error: %v", err)
	}
	
	// Test RemoveRule
	manager.RemoveRule("manager-test-rule")
	
	if len(manager.Config().Rules) != 0 {
		t.Errorf("Expected 0 rules after RemoveRule, got %d", len(manager.Config().Rules))
	}
	
	// Test removing non-existent rule (should not panic)
	manager.RemoveRule("non-existent")
}

func TestLoadInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.json")
	
	// Write invalid JSON
	if err := os.WriteFile(configPath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}
	
	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() should return error for invalid JSON")
	}
}

func TestSaveWithMissingDirectory(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "missing", "dir", "config.json")
	
	cfg := New()
	if err := Save(cfg, configPath); err != nil {
		t.Errorf("Save() should create missing directories, got error: %v", err)
	}
	
	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created in missing directory")
	}
}
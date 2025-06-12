package models

import (
	"testing"
	"time"
)

func TestFileFormatString(t *testing.T) {
	tests := []struct {
		format   FileFormat
		expected string
	}{
		{FormatJSON, "json"},
		{FormatYAML, "yaml"},
		{FormatTOML, "toml"},
	}
	
	for _, test := range tests {
		if test.format.String() != test.expected {
			t.Errorf("FileFormat.String() = %s, expected %s", test.format.String(), test.expected)
		}
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		filepath string
		expected FileFormat
	}{
		{"config.json", FormatJSON},
		{"config.yaml", FormatYAML},
		{"config.yml", FormatYAML},
		{"config.toml", FormatTOML},
		{"config.txt", FormatJSON}, // default
		{"config", FormatJSON},     // default
		{"/path/to/config.yaml", FormatYAML},
		{"/path/to/config.json", FormatJSON},
		{"", FormatJSON}, // default for empty string
		{"file.JSON", FormatJSON}, // case sensitive
		{"file.YAML", FormatJSON}, // case sensitive, should default to JSON
	}
	
	for _, test := range tests {
		result := DetectFormat(test.filepath)
		if result != test.expected {
			t.Errorf("DetectFormat(%s) = %s, expected %s", test.filepath, result, test.expected)
		}
	}
}

func TestSyncRuleStruct(t *testing.T) {
	now := time.Now()
	lastSync := now.Add(-time.Hour)
	
	rule := SyncRule{
		ID:          "test-rule-1",
		Name:        "Test Rule",
		Description: "A test sync rule",
		SourceFile:  "source.yaml",
		SourceKey:   "database.host",
		TargetFile:  "target.json",
		TargetKey:   "config.db.host",
		Enabled:     true,
		Created:     now,
		LastSync:    &lastSync,
	}
	
	// Test all fields are set correctly
	if rule.ID != "test-rule-1" {
		t.Errorf("Expected ID 'test-rule-1', got %s", rule.ID)
	}
	
	if rule.Name != "Test Rule" {
		t.Errorf("Expected Name 'Test Rule', got %s", rule.Name)
	}
	
	if rule.Description != "A test sync rule" {
		t.Errorf("Expected Description 'A test sync rule', got %s", rule.Description)
	}
	
	if rule.SourceFile != "source.yaml" {
		t.Errorf("Expected SourceFile 'source.yaml', got %s", rule.SourceFile)
	}
	
	if rule.SourceKey != "database.host" {
		t.Errorf("Expected SourceKey 'database.host', got %s", rule.SourceKey)
	}
	
	if rule.TargetFile != "target.json" {
		t.Errorf("Expected TargetFile 'target.json', got %s", rule.TargetFile)
	}
	
	if rule.TargetKey != "config.db.host" {
		t.Errorf("Expected TargetKey 'config.db.host', got %s", rule.TargetKey)
	}
	
	if !rule.Enabled {
		t.Error("Expected Enabled to be true")
	}
	
	if rule.Created != now {
		t.Errorf("Expected Created time %v, got %v", now, rule.Created)
	}
	
	if rule.LastSync == nil {
		t.Error("Expected LastSync to be set")
	} else if *rule.LastSync != lastSync {
		t.Errorf("Expected LastSync time %v, got %v", lastSync, *rule.LastSync)
	}
}

func TestSyncEventStruct(t *testing.T) {
	now := time.Now()
	
	event := SyncEvent{
		RuleID:    "test-rule-1",
		Timestamp: now,
		OldValue:  "old_value",
		NewValue:  "new_value",
		Success:   true,
		Error:     "",
	}
	
	// Test all fields are set correctly
	if event.RuleID != "test-rule-1" {
		t.Errorf("Expected RuleID 'test-rule-1', got %s", event.RuleID)
	}
	
	if event.Timestamp != now {
		t.Errorf("Expected Timestamp %v, got %v", now, event.Timestamp)
	}
	
	if event.OldValue != "old_value" {
		t.Errorf("Expected OldValue 'old_value', got %v", event.OldValue)
	}
	
	if event.NewValue != "new_value" {
		t.Errorf("Expected NewValue 'new_value', got %v", event.NewValue)
	}
	
	if !event.Success {
		t.Error("Expected Success to be true")
	}
	
	if event.Error != "" {
		t.Errorf("Expected Error to be empty, got %s", event.Error)
	}
}

func TestSyncEventWithError(t *testing.T) {
	now := time.Now()
	
	event := SyncEvent{
		RuleID:    "test-rule-1",
		Timestamp: now,
		OldValue:  "old_value",
		NewValue:  nil,
		Success:   false,
		Error:     "sync failed: file not found",
	}
	
	if event.Success {
		t.Error("Expected Success to be false")
	}
	
	if event.Error != "sync failed: file not found" {
		t.Errorf("Expected Error 'sync failed: file not found', got %s", event.Error)
	}
	
	if event.NewValue != nil {
		t.Errorf("Expected NewValue to be nil, got %v", event.NewValue)
	}
}

func TestConfigStruct(t *testing.T) {
	rule1 := SyncRule{
		ID:         "rule-1",
		Name:       "Rule 1",
		SourceFile: "source1.yaml",
		SourceKey:  "key1",
		TargetFile: "target1.json",
		TargetKey:  "key1",
		Enabled:    true,
		Created:    time.Now(),
	}
	
	rule2 := SyncRule{
		ID:         "rule-2",
		Name:       "Rule 2",
		SourceFile: "source2.toml",
		SourceKey:  "key2",
		TargetFile: "target2.json",
		TargetKey:  "key2",
		Enabled:    false,
		Created:    time.Now(),
	}
	
	config := Config{
		Rules:   []SyncRule{rule1, rule2},
		LogFile: "var-sync.log",
		Debug:   true,
	}
	
	if len(config.Rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(config.Rules))
	}
	
	if config.Rules[0].ID != "rule-1" {
		t.Errorf("Expected first rule ID 'rule-1', got %s", config.Rules[0].ID)
	}
	
	if config.Rules[1].ID != "rule-2" {
		t.Errorf("Expected second rule ID 'rule-2', got %s", config.Rules[1].ID)
	}
	
	if config.LogFile != "var-sync.log" {
		t.Errorf("Expected LogFile 'var-sync.log', got %s", config.LogFile)
	}
	
	if !config.Debug {
		t.Error("Expected Debug to be true")
	}
}

func TestConfigWithEmptyRules(t *testing.T) {
	config := Config{
		Rules:   []SyncRule{},
		LogFile: "test.log",
		Debug:   false,
	}
	
	if len(config.Rules) != 0 {
		t.Errorf("Expected 0 rules, got %d", len(config.Rules))
	}
	
	if config.LogFile != "test.log" {
		t.Errorf("Expected LogFile 'test.log', got %s", config.LogFile)
	}
	
	if config.Debug {
		t.Error("Expected Debug to be false")
	}
}

func TestSyncRuleWithNilLastSync(t *testing.T) {
	rule := SyncRule{
		ID:         "test-rule",
		Name:       "Test Rule",
		SourceFile: "source.yaml",
		SourceKey:  "key",
		TargetFile: "target.json",
		TargetKey:  "key",
		Enabled:    true,
		Created:    time.Now(),
		LastSync:   nil, // explicitly set to nil
	}
	
	if rule.LastSync != nil {
		t.Error("Expected LastSync to be nil")
	}
}

func TestSyncEventWithComplexValues(t *testing.T) {
	now := time.Now()
	
	oldValue := map[string]interface{}{
		"host": "localhost",
		"port": 5432,
	}
	
	newValue := map[string]interface{}{
		"host": "production.db.com",
		"port": 5432,
	}
	
	event := SyncEvent{
		RuleID:    "complex-rule",
		Timestamp: now,
		OldValue:  oldValue,
		NewValue:  newValue,
		Success:   true,
		Error:     "",
	}
	
	if event.OldValue == nil {
		t.Error("Expected OldValue to be set")
	}
	
	if event.NewValue == nil {
		t.Error("Expected NewValue to be set")
	}
	
	// Test that we can access the complex values
	if oldMap, ok := event.OldValue.(map[string]interface{}); ok {
		if oldMap["host"] != "localhost" {
			t.Errorf("Expected old host 'localhost', got %v", oldMap["host"])
		}
	} else {
		t.Error("Expected OldValue to be a map")
	}
	
	if newMap, ok := event.NewValue.(map[string]interface{}); ok {
		if newMap["host"] != "production.db.com" {
			t.Errorf("Expected new host 'production.db.com', got %v", newMap["host"])
		}
	} else {
		t.Error("Expected NewValue to be a map")
	}
}
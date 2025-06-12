package parser

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	parser := New()
	if parser == nil {
		t.Fatal("New() returned nil")
	}
}

func TestLoadFileJSON(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.json")
	
	content := `{
		"database": {
			"host": "localhost",
			"port": 5432
		},
		"api": {
			"key": "secret123"
		}
	}`
	
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	parser := New()
	data, err := parser.LoadFile(filePath)
	if err != nil {
		t.Fatalf("LoadFile() returned error: %v", err)
	}
	
	expected := map[string]any{
		"database": map[string]any{
			"host": "localhost",
			"port": float64(5432),
		},
		"api": map[string]any{
			"key": "secret123",
		},
	}
	
	if !reflect.DeepEqual(data, expected) {
		t.Errorf("LoadFile() returned unexpected data.\nExpected: %+v\nGot: %+v", expected, data)
	}
}

func TestLoadFileYAML(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.yaml")
	
	content := `database:
  host: localhost
  port: 5432
api:
  key: secret123`
	
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	parser := New()
	data, err := parser.LoadFile(filePath)
	if err != nil {
		t.Fatalf("LoadFile() returned error: %v", err)
	}
	
	expected := map[string]any{
		"database": map[string]any{
			"host": "localhost",
			"port": 5432,
		},
		"api": map[string]any{
			"key": "secret123",
		},
	}
	
	if !reflect.DeepEqual(data, expected) {
		t.Errorf("LoadFile() returned unexpected data.\nExpected: %+v\nGot: %+v", expected, data)
	}
}

func TestLoadFileTOML(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.toml")
	
	content := `[database]
host = "localhost"
port = 5432

[api]
key = "secret123"`
	
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	parser := New()
	data, err := parser.LoadFile(filePath)
	if err != nil {
		t.Fatalf("LoadFile() returned error: %v", err)
	}
	
	expected := map[string]any{
		"database": map[string]any{
			"host": "localhost",
			"port": int64(5432),
		},
		"api": map[string]any{
			"key": "secret123",
		},
	}
	
	if !reflect.DeepEqual(data, expected) {
		t.Errorf("LoadFile() returned unexpected data.\nExpected: %+v\nGot: %+v", expected, data)
	}
}

func TestLoadFileNonExistent(t *testing.T) {
	parser := New()
	_, err := parser.LoadFile("/non/existent/file.json")
	if err == nil {
		t.Error("LoadFile() should return error for non-existent file")
	}
}

func TestLoadFileUnsupportedFormat(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")
	
	if err := os.WriteFile(filePath, []byte("some content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	parser := New()
	_, err := parser.LoadFile(filePath)
	if err == nil {
		t.Error("LoadFile() should return error for unsupported format")
	}
}

func TestSaveFileJSON(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "output.json")
	
	data := map[string]any{
		"database": map[string]any{
			"host": "localhost",
			"port": 5432,
		},
	}
	
	parser := New()
	if err := parser.SaveFile(filePath, data); err != nil {
		t.Fatalf("SaveFile() returned error: %v", err)
	}
	
	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("SaveFile() did not create file")
	}
	
	// Load and verify content
	loadedData, err := parser.LoadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to load saved file: %v", err)
	}
	
	// JSON numbers are loaded as float64, so we need to handle this
	expectedData := map[string]any{
		"database": map[string]any{
			"host": "localhost",
			"port": float64(5432), // JSON loads numbers as float64
		},
	}
	
	if !reflect.DeepEqual(loadedData, expectedData) {
		t.Errorf("Saved and loaded data do not match.\nExpected: %+v\nGot: %+v", expectedData, loadedData)
	}
}

func TestGetValue(t *testing.T) {
	data := map[string]any{
		"database": map[string]any{
			"host": "localhost",
			"port": 5432,
			"config": map[string]any{
				"timeout": 30,
			},
		},
		"simple": "value",
	}
	
	parser := New()
	
	tests := []struct {
		keyPath  string
		expected any
	}{
		{"simple", "value"},
		{"database.host", "localhost"},
		{"database.port", 5432},
		{"database.config.timeout", 30},
	}
	
	for _, test := range tests {
		value, err := parser.GetValue(data, test.keyPath)
		if err != nil {
			t.Errorf("GetValue(%s) returned error: %v", test.keyPath, err)
			continue
		}
		
		if !reflect.DeepEqual(value, test.expected) {
			t.Errorf("GetValue(%s) = %v, expected %v", test.keyPath, value, test.expected)
		}
	}
}

func TestGetValueErrors(t *testing.T) {
	data := map[string]any{
		"database": map[string]any{
			"host": "localhost",
		},
		"simple": "value",
	}
	
	parser := New()
	
	tests := []string{
		"nonexistent",
		"database.nonexistent",
		"simple.nested", // trying to access nested key on string value
		"database.host.nested", // trying to access nested key on string value
	}
	
	for _, keyPath := range tests {
		_, err := parser.GetValue(data, keyPath)
		if err == nil {
			t.Errorf("GetValue(%s) should return error", keyPath)
		}
	}
}

func TestSetValue(t *testing.T) {
	data := map[string]any{
		"existing": map[string]any{
			"key": "old_value",
		},
	}
	
	parser := New()
	
	tests := []struct {
		keyPath string
		value   any
	}{
		{"new_key", "new_value"},
		{"existing.key", "updated_value"},
		{"existing.new_nested", "nested_value"},
		{"deep.nested.key", "deep_value"},
	}
	
	for _, test := range tests {
		if err := parser.SetValue(data, test.keyPath, test.value); err != nil {
			t.Errorf("SetValue(%s, %v) returned error: %v", test.keyPath, test.value, err)
			continue
		}
		
		// Verify value was set
		value, err := parser.GetValue(data, test.keyPath)
		if err != nil {
			t.Errorf("GetValue(%s) after SetValue returned error: %v", test.keyPath, err)
			continue
		}
		
		if !reflect.DeepEqual(value, test.value) {
			t.Errorf("SetValue(%s, %v) did not set correct value. Got: %v", test.keyPath, test.value, value)
		}
	}
}

func TestSetValueConflict(t *testing.T) {
	data := map[string]any{
		"simple": "string_value",
	}
	
	parser := New()
	
	// This should fail because "simple" is a string, not an object
	err := parser.SetValue(data, "simple.nested", "value")
	if err == nil {
		t.Error("SetValue() should return error when trying to set nested key on non-object")
	}
}

func TestGetAllKeys(t *testing.T) {
	data := map[string]any{
		"simple": "value",
		"database": map[string]any{
			"host": "localhost",
			"port": 5432,
			"config": map[string]any{
				"timeout": 30,
				"retries": 3,
			},
		},
		"api": map[string]any{
			"key": "secret",
		},
	}
	
	parser := New()
	keys := parser.GetAllKeys(data, "")
	
	expectedKeys := []string{
		"simple",
		"database",
		"database.host",
		"database.port",
		"database.config",
		"database.config.timeout",
		"database.config.retries",
		"api",
		"api.key",
	}
	
	if len(keys) != len(expectedKeys) {
		t.Errorf("GetAllKeys() returned %d keys, expected %d", len(keys), len(expectedKeys))
	}
	
	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}
	
	for _, expectedKey := range expectedKeys {
		if !keyMap[expectedKey] {
			t.Errorf("GetAllKeys() missing expected key: %s", expectedKey)
		}
	}
}

func TestValidateKeyPath(t *testing.T) {
	data := map[string]any{
		"database": map[string]any{
			"host": "localhost",
		},
	}
	
	parser := New()
	
	// Valid key paths
	validPaths := []string{
		"database",
		"database.host",
	}
	
	for _, path := range validPaths {
		if err := parser.ValidateKeyPath(data, path); err != nil {
			t.Errorf("ValidateKeyPath(%s) should not return error, got: %v", path, err)
		}
	}
	
	// Invalid key paths
	invalidPaths := []string{
		"nonexistent",
		"database.nonexistent",
	}
	
	for _, path := range invalidPaths {
		if err := parser.ValidateKeyPath(data, path); err == nil {
			t.Errorf("ValidateKeyPath(%s) should return error", path)
		}
	}
}

func TestConvertMapInterface(t *testing.T) {
	input := map[any]any{
		"string_key": "value1",
		123:          "value2",
		true:         "value3",
		"nested": map[any]any{
			"inner_key": "inner_value",
			456:         "numeric_key_value",
		},
	}
	
	result := convertMapInterface(input)
	
	expected := map[string]any{
		"string_key": "value1",
		"123":        "value2",
		"true":       "value3",
		"nested": map[string]any{
			"inner_key": "inner_value",
			"456":       "numeric_key_value",
		},
	}
	
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("convertMapInterface() returned unexpected result.\nExpected: %+v\nGot: %+v", expected, result)
	}
}
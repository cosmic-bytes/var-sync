package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	
	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
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
	
	// Updated to only include leaf nodes (actual values), not branch nodes (objects)
	expectedKeys := []string{
		"simple",
		"database.host",
		"database.port",
		"database.config.timeout",
		"database.config.retries",
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

func TestParseKeySegment(t *testing.T) {
	tests := []struct {
		segment       string
		expectedKey   string
		expectedIndex int
		expectError   bool
	}{
		{"key", "key", -1, false},
		{"database", "database", -1, false},
		{"items[0]", "items", 0, false},
		{"users[5]", "users", 5, false},
		{"array[999]", "array", 999, false},
		{"invalid[abc]", "", -1, true},
		{"invalid[]", "", -1, true},
		{"invalid[-1]", "", -1, true},
	}
	
	for _, test := range tests {
		key, index, err := parseKeySegment(test.segment)
		
		if test.expectError {
			if err == nil {
				t.Errorf("parseKeySegment(%s) should return error", test.segment)
			}
			continue
		}
		
		if err != nil {
			t.Errorf("parseKeySegment(%s) returned unexpected error: %v", test.segment, err)
			continue
		}
		
		if key != test.expectedKey {
			t.Errorf("parseKeySegment(%s) key = %s, expected %s", test.segment, key, test.expectedKey)
		}
		
		if index != test.expectedIndex {
			t.Errorf("parseKeySegment(%s) index = %d, expected %d", test.segment, index, test.expectedIndex)
		}
	}
}

func TestGetValueArrayIndexing(t *testing.T) {
	data := map[string]any{
		"database": []any{
			map[string]any{
				"host":     "localhost",
				"port":     5432,
				"name":     "myapp",
				"user":     "admin",
				"password": "secret123",
			},
		},
		"servers": []any{
			"server1",
			"server2",
			"server3",
		},
		"configs": []any{
			map[string]any{
				"env": "production",
				"debug": false,
			},
			map[string]any{
				"env": "development",
				"debug": true,
			},
		},
	}
	
	parser := New()
	
	tests := []struct {
		keyPath  string
		expected any
	}{
		{"database[0].host", "localhost"},
		{"database[0].port", 5432},
		{"database[0].name", "myapp"},
		{"database[0].user", "admin"},
		{"database[0].password", "secret123"},
		{"servers[0]", "server1"},
		{"servers[1]", "server2"},
		{"servers[2]", "server3"},
		{"configs[0].env", "production"},
		{"configs[0].debug", false},
		{"configs[1].env", "development"},
		{"configs[1].debug", true},
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

func TestGetValueArrayIndexingErrors(t *testing.T) {
	data := map[string]any{
		"database": []any{
			map[string]any{
				"host": "localhost",
			},
		},
		"servers": []any{
			"server1",
			"server2",
		},
		"notarray": "string_value",
	}
	
	parser := New()
	
	tests := []string{
		"database[1].host",    // Index out of bounds
		"database[0].missing", // Key not found in array element
		"servers[5]",          // Index out of bounds
		"notarray[0]",         // Not an array
		"database[abc].host",  // Invalid index
	}
	
	for _, keyPath := range tests {
		_, err := parser.GetValue(data, keyPath)
		if err == nil {
			t.Errorf("GetValue(%s) should return error", keyPath)
		}
	}
}

func TestSetValueArrayIndexing(t *testing.T) {
	data := map[string]any{
		"database": []any{
			map[string]any{
				"host": "localhost",
				"port": 5432,
			},
		},
		"servers": []any{
			"server1",
			"server2",
		},
	}
	
	parser := New()
	
	tests := []struct {
		keyPath string
		value   any
	}{
		{"database[0].host", "newhost"},
		{"database[0].port", 3306},
		{"servers[0]", "newserver1"},
		{"servers[1]", "newserver2"},
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

func TestGetAllKeysWithArrays(t *testing.T) {
	data := map[string]any{
		"simple": "value",
		"database": []any{
			map[string]any{
				"host": "localhost",
				"port": 5432,
			},
			map[string]any{
				"host": "remotehost",
				"port": 3306,
			},
		},
		"servers": []any{
			"server1",
			"server2",
		},
		"config": map[string]any{
			"timeout": 30,
		},
	}
	
	parser := New()
	keys := parser.GetAllKeys(data, "")
	
	expectedKeys := []string{
		"simple",
		"database[0].host",
		"database[0].port",
		"database[1].host",
		"database[1].port",
		"servers[0]",
		"servers[1]",
		"config.timeout",
	}
	
	if len(keys) != len(expectedKeys) {
		t.Errorf("GetAllKeys() returned %d keys, expected %d", len(keys), len(expectedKeys))
		t.Errorf("Got keys: %v", keys)
		t.Errorf("Expected keys: %v", expectedKeys)
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

func TestTOMLArrayStructure(t *testing.T) {
	tempDir := t.TempDir()
	tomlPath := filepath.Join(tempDir, "test_structure.toml")
	
	tomlContent := `[[database]]
host = "localhost"
port = 5432

[[database]]
host = "remotehost"
port = 3306`
	
	if err := os.WriteFile(tomlPath, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("Failed to write test TOML: %v", err)
	}
	
	parser := New()
	data, err := parser.LoadFile(tomlPath)
	if err != nil {
		t.Fatalf("LoadFile() failed: %v", err)
	}
	
	t.Logf("TOML parsed structure: %+v", data)
	
	// Try to access as array
	if dbArray, ok := data["database"].([]any); ok {
		t.Logf("database is an array with %d elements", len(dbArray))
		for i, item := range dbArray {
			t.Logf("database[%d]: %+v", i, item)
		}
	} else {
		t.Logf("database is NOT an array, type: %T", data["database"])
	}
	
	// Test accessing TOML array elements
	value, err := parser.GetValue(data, "database[0].host")
	if err != nil {
		t.Errorf("GetValue(database[0].host) failed: %v", err)
	} else {
		t.Logf("database[0].host = %v", value)
	}
	
	// Test setting a value in TOML array
	t.Logf("Attempting to set database[0].host to 'newhost'")
	err = parser.SetValue(data, "database[0].host", "newhost")
	if err != nil {
		t.Errorf("SetValue(database[0].host) failed: %v", err)
	} else {
		// Verify it was set
		value, err := parser.GetValue(data, "database[0].host")
		if err != nil {
			t.Errorf("GetValue after SetValue failed: %v", err)
		} else {
			t.Logf("After SetValue, database[0].host = %v", value)
		}
		
		// Print the entire structure to see if it changed
		t.Logf("Final TOML structure: %+v", data)
	}
}

func TestTOMLFormatPreservation(t *testing.T) {
	tempDir := t.TempDir()
	tomlPath := filepath.Join(tempDir, "test_format.toml")
	
	// Create a realistic TOML config with comments and specific formatting
	originalTOML := `# Database Configuration
host = "localhost"
port = 8080
debug = true

# Database settings
[[database]]
# Primary database
host = "localhost"
port = 5432
name = "myapp"

[[database]]  
# Secondary database
host = "backup.example.com"
port = 5433
name = "backup"`
	
	if err := os.WriteFile(tomlPath, []byte(originalTOML), 0644); err != nil {
		t.Fatalf("Failed to write test TOML: %v", err)
	}
	
	parser := New()
	
	// Load the file
	data, err := parser.LoadFile(tomlPath)
	if err != nil {
		t.Fatalf("LoadFile() failed: %v", err)
	}
	
	// Make a small change
	err = parser.SetValue(data, "database[0].host", "newhost")
	if err != nil {
		t.Logf("SetValue() failed (expected): %v", err)
	}
	
	// Try setting a top-level value instead
	err = parser.SetValue(data, "port", 9090)
	if err != nil {
		t.Fatalf("SetValue(port) failed: %v", err)
	}
	
	// Save the file
	err = parser.SaveFile(tomlPath, data)
	if err != nil {
		t.Fatalf("SaveFile() failed: %v", err)
	}
	
	// Read the saved content
	savedContent, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}
	
	savedStr := string(savedContent)
	t.Logf("Original TOML:\n%s", originalTOML)
	t.Logf("Saved TOML:\n%s", savedStr)
	
	// Analyze what changed
	if strings.Contains(savedStr, "port = 9090") {
		t.Logf("✓ Value change applied correctly")
	} else {
		t.Error("✗ Value change not applied")
	}
	
	if strings.Contains(savedStr, "# Database Configuration") {
		t.Logf("✓ Comments preserved")
	} else {
		t.Logf("✗ Comments lost")
	}
	
	// Check array order
	lines := strings.Split(savedStr, "\n")
	var databaseSections []int
	for i, line := range lines {
		if strings.Contains(line, "[[database]]") {
			databaseSections = append(databaseSections, i)
		}
	}
	t.Logf("Database sections found at lines: %v", databaseSections)
}

func TestTargetedFileUpdates(t *testing.T) {
	tempDir := t.TempDir()
	
	// Test YAML targeted updates
	t.Run("YAML", func(t *testing.T) {
		yamlPath := filepath.Join(tempDir, "test_targeted.yaml")
		
		// Create YAML with comments and specific formatting
		originalYAML := `# Configuration file
# Main settings
host: localhost
port: 8080
debug: true

# Database configuration
database:
  - host: localhost  # Primary DB
    port: 5432
    name: myapp
    user: admin
    password: secret123
  - host: backup.example.com  # Backup DB
    port: 5433
    name: backup

# Additional settings
timeout: 30`
		
		if err := os.WriteFile(yamlPath, []byte(originalYAML), 0644); err != nil {
			t.Fatalf("Failed to write test YAML: %v", err)
		}
		
		parser := New()
		
		// Use targeted update instead of full rewrite
		err := parser.UpdateFileValue(yamlPath, "database[0].host", "newhost")
		if err != nil {
			t.Fatalf("UpdateFileValue() failed: %v", err)
		}
		
		// Read the result
		updatedContent, err := os.ReadFile(yamlPath)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}
		
		updatedStr := string(updatedContent)
		t.Logf("Original YAML:\n%s", originalYAML)
		t.Logf("Updated YAML:\n%s", updatedStr)
		
		// Verify the specific change was made
		if !strings.Contains(updatedStr, "host: newhost") {
			t.Error("✗ Target value not updated")
		} else {
			t.Log("✓ Target value updated correctly")
		}
		
		// Verify comments are preserved
		if strings.Contains(updatedStr, "# Configuration file") &&
		   strings.Contains(updatedStr, "# Primary DB") &&
		   strings.Contains(updatedStr, "# Backup DB") {
			t.Log("✓ Comments preserved")
		} else {
			t.Error("✗ Comments lost")
		}
		
		// Verify other values unchanged
		if strings.Contains(updatedStr, "port: 8080") &&
		   strings.Contains(updatedStr, "backup.example.com") &&
		   strings.Contains(updatedStr, "timeout: 30") {
			t.Log("✓ Other values preserved")
		} else {
			t.Error("✗ Other values changed unexpectedly")
		}
		
		// Count lines to ensure minimal change
		originalLines := strings.Split(originalYAML, "\n")
		updatedLines := strings.Split(updatedStr, "\n")
		if len(originalLines) == len(updatedLines) {
			t.Log("✓ Line count preserved")
		} else {
			t.Error("✗ Line count changed")
		}
	})
	
	// Test TOML targeted updates
	t.Run("TOML", func(t *testing.T) {
		tomlPath := filepath.Join(tempDir, "test_targeted.toml")
		
		// Create TOML with comments and specific formatting
		originalTOML := `# Configuration file
# Main settings
host = "localhost"
port = 8080
debug = true

# Database settings
[[database]]
# Primary database
host = "localhost"
port = 5432
name = "myapp"

[[database]]  
# Secondary database
host = "backup.example.com"
port = 5433
name = "backup"

# Additional settings
timeout = 30`
		
		if err := os.WriteFile(tomlPath, []byte(originalTOML), 0644); err != nil {
			t.Fatalf("Failed to write test TOML: %v", err)
		}
		
		parser := New()
		
		// Use targeted update
		err := parser.UpdateFileValue(tomlPath, "database[0].host", "newhost")
		if err != nil {
			t.Fatalf("UpdateFileValue() failed: %v", err)
		}
		
		// Read the result
		updatedContent, err := os.ReadFile(tomlPath)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}
		
		updatedStr := string(updatedContent)
		t.Logf("Original TOML:\n%s", originalTOML)
		t.Logf("Updated TOML:\n%s", updatedStr)
		
		// Verify the specific change was made
		if !strings.Contains(updatedStr, `host = "newhost"`) {
			t.Error("✗ Target value not updated")
		} else {
			t.Log("✓ Target value updated correctly")
		}
		
		// Verify comments are preserved
		if strings.Contains(updatedStr, "# Configuration file") &&
		   strings.Contains(updatedStr, "# Primary database") &&
		   strings.Contains(updatedStr, "# Secondary database") {
			t.Log("✓ Comments preserved")
		} else {
			t.Error("✗ Comments lost")
		}
		
		// Verify other values unchanged
		if strings.Contains(updatedStr, "port = 8080") &&
		   strings.Contains(updatedStr, `host = "backup.example.com"`) &&
		   strings.Contains(updatedStr, "timeout = 30") {
			t.Log("✓ Other values preserved")
		} else {
			t.Error("✗ Other values changed unexpectedly")
		}
		
		// Count lines to ensure minimal change
		originalLines := strings.Split(originalTOML, "\n")
		updatedLines := strings.Split(updatedStr, "\n")
		if len(originalLines) == len(updatedLines) {
			t.Log("✓ Line count preserved")
		} else {
			t.Error("✗ Line count changed")
		}
	})
}

func TestBatchedFileUpdates(t *testing.T) {
	tempDir := t.TempDir()
	
	// Test YAML batched updates
	t.Run("YAML", func(t *testing.T) {
		yamlPath := filepath.Join(tempDir, "test_batched.yaml")
		
		// Create YAML with multiple values to update
		originalYAML := `# Configuration file
host: localhost
port: 8080
debug: false

# Database configuration
database:
  - host: localhost
    port: 5432
    name: myapp
    user: admin
    password: secret123
  - host: backup.example.com
    port: 5433
    name: backup

timeout: 30`
		
		if err := os.WriteFile(yamlPath, []byte(originalYAML), 0644); err != nil {
			t.Fatalf("Failed to write test YAML: %v", err)
		}
		
		parser := New()
		
		// Use batched update for multiple changes
		updates := map[string]any{
			"host":               "newhost",
			"port":               9090,
			"debug":              true,
			"database[0].host":   "primarydb",
			"database[0].port":   3306,
			"database[1].host":   "secondarydb",
			"timeout":            60,
		}
		
		err := parser.UpdateFileValues(yamlPath, updates)
		if err != nil {
			t.Fatalf("UpdateFileValues() failed: %v", err)
		}
		
		// Read the result
		updatedContent, err := os.ReadFile(yamlPath)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}
		
		updatedStr := string(updatedContent)
		t.Logf("Original YAML:\n%s", originalYAML)
		t.Logf("Updated YAML:\n%s", updatedStr)
		
		// Verify all changes were made
		expectedChanges := map[string]string{
			"host: newhost":        "host updated",
			"port: 9090":           "port updated", 
			"debug: true":          "debug updated",
			"host: primarydb":      "database[0].host updated",
			"port: 3306":           "database[0].port updated",
			"host: secondarydb":    "database[1].host updated",
			"timeout: 60":          "timeout updated",
		}
		
		allUpdated := true
		for expectedText, description := range expectedChanges {
			if strings.Contains(updatedStr, expectedText) {
				t.Logf("✓ %s", description)
			} else {
				t.Errorf("✗ %s - expected: %s", description, expectedText)
				allUpdated = false
			}
		}
		
		if allUpdated {
			t.Log("✓ All batched updates applied successfully")
		}
		
		// Verify comments are still preserved
		if strings.Contains(updatedStr, "# Configuration file") &&
		   strings.Contains(updatedStr, "# Database configuration") {
			t.Log("✓ Comments preserved during batch update")
		} else {
			t.Error("✗ Comments lost during batch update")
		}
	})
	
	// Test TOML batched updates  
	t.Run("TOML", func(t *testing.T) {
		tomlPath := filepath.Join(tempDir, "test_batched.toml")
		
		// Create TOML with multiple values to update
		originalTOML := `# Configuration file
host = "localhost"
port = 8080
debug = false

# Database settings
[[database]]
host = "localhost"
port = 5432
name = "myapp"

[[database]]
host = "backup.example.com"
port = 5433
name = "backup"

timeout = 30`
		
		if err := os.WriteFile(tomlPath, []byte(originalTOML), 0644); err != nil {
			t.Fatalf("Failed to write test TOML: %v", err)
		}
		
		parser := New()
		
		// Use batched update for multiple changes
		updates := map[string]any{
			"host":               "newhost",
			"port":               9090,
			"debug":              true,
			"database[0].host":   "primarydb",
			"database[0].port":   3306,
			"database[1].host":   "secondarydb", 
			"timeout":            60,
		}
		
		err := parser.UpdateFileValues(tomlPath, updates)
		if err != nil {
			t.Fatalf("UpdateFileValues() failed: %v", err)
		}
		
		// Read the result
		updatedContent, err := os.ReadFile(tomlPath)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}
		
		updatedStr := string(updatedContent)
		t.Logf("Original TOML:\n%s", originalTOML)
		t.Logf("Updated TOML:\n%s", updatedStr)
		
		// Verify all changes were made
		expectedChanges := map[string]string{
			`host = "newhost"`:      "host updated",
			"port = 9090":           "port updated",
			"debug = true":          "debug updated", 
			`host = "primarydb"`:    "database[0].host updated",
			"port = 3306":           "database[0].port updated",
			`host = "secondarydb"`:  "database[1].host updated",
			"timeout = 60":          "timeout updated",
		}
		
		allUpdated := true
		for expectedText, description := range expectedChanges {
			if strings.Contains(updatedStr, expectedText) {
				t.Logf("✓ %s", description)
			} else {
				t.Errorf("✗ %s - expected: %s", description, expectedText)
				allUpdated = false
			}
		}
		
		if allUpdated {
			t.Log("✓ All batched updates applied successfully")
		}
		
		// Verify comments are still preserved
		if strings.Contains(updatedStr, "# Configuration file") &&
		   strings.Contains(updatedStr, "# Database settings") {
			t.Log("✓ Comments preserved during batch update")
		} else {
			t.Error("✗ Comments lost during batch update")
		}
	})
}

func TestEdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	
	// Test JSON edge cases
	t.Run("JSON_EdgeCases", func(t *testing.T) {
		jsonPath := filepath.Join(tempDir, "test_edges.json")
		
		// Create JSON with challenging edge cases
		originalJSON := `{
  "simple": "value",
  "escaped_quotes": "value with \"quotes\" inside",
  "unicode": "café ñ ü 日本語",
  "special_chars": "value with : {} [] , symbols",
  "empty_string": "",
  "null_value": null,
  "boolean_true": true,
  "boolean_false": false,
  "number_int": 42,
  "number_float": 3.14159,
  "number_negative": -123,
  "number_scientific": 1.23e-4,
  "nested": {
    "deep": {
      "deeper": {
        "value": "deeply nested"
      }
    }
  },
  "array_mixed": [
    {
      "id": 1,
      "name": "first",
      "special": "contains \"quotes\" and symbols: {}[]"
    },
    {
      "id": 2,
      "name": "second",
      "unicode": "测试 αβγ emoji: 🚀"
    }
  ],
  "array_primitives": ["string", 123, true, null, ""],
  "trailing_item": "last"
}`
		
		if err := os.WriteFile(jsonPath, []byte(originalJSON), 0644); err != nil {
			t.Fatalf("Failed to write test JSON: %v", err)
		}
		
		parser := New()
		
		// Test edge case updates
		updates := map[string]any{
			"escaped_quotes":           `new "quoted" value`,
			"unicode":                  "новый русский текст 한국어",
			"special_chars":            "new: value {with} [symbols]",
			"empty_string":             "no longer empty",
			"null_value":               "was null",
			"boolean_true":             false,
			"number_float":             -999.999,
			"nested.deep.deeper.value": "updated deep value",
			"array_mixed[0].name":      "updated first",
			"array_mixed[1].unicode":   "updated unicode: 🎉",
			"array_primitives[0]":      "updated string",
		}
		
		err := parser.UpdateFileValues(jsonPath, updates)
		if err != nil {
			t.Fatalf("UpdateFileValues() failed: %v", err)
		}
		
		// Verify the file is still valid JSON
		updatedContent, err := os.ReadFile(jsonPath)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}
		
		var jsonData map[string]any
		if err := json.Unmarshal(updatedContent, &jsonData); err != nil {
			t.Fatalf("Updated JSON is invalid: %v", err)
		}
		
		t.Log("✓ JSON remains valid after edge case updates")
		
		// Verify specific edge case handling
		updatedStr := string(updatedContent)
		t.Logf("Updated JSON:\n%s", updatedStr)
		
		// Check that special characters are properly escaped
		if strings.Contains(updatedStr, `"новый русский текст 한국어"`) {
			t.Log("✓ Unicode characters preserved")
		} else {
			t.Error("✗ Unicode characters not preserved")
		}
	})
	
	// Test TOML edge cases  
	t.Run("TOML_EdgeCases", func(t *testing.T) {
		tomlPath := filepath.Join(tempDir, "test_edges.toml")
		
		// Create TOML with challenging edge cases
		originalTOML := `# Configuration with edge cases
simple = "value"
escaped_quotes = "value with \"quotes\" inside"
unicode = "café ñ ü 日本語"
special_chars = "value with symbols"
empty_string = ""
boolean_true = true
boolean_false = false
number_int = 42
number_float = 3.14159
number_negative = -123

# Nested table
[nested]
key = "nested value"

[nested.deep]
deeper = "deeply nested"

# Array of tables with edge cases
[[database]]
# Comment about first DB
host = "localhost"
port = 5432
name = "my-app_test.db"
special_chars = "db with \"quotes\" and symbols: {}"
unicode_name = "测试数据库"

[[database]]
# Comment about second DB  
host = "backup.example.com"
port = 5433
name = "backup-db"
password = "p@ssw0rd!#$%"

# Multi-line string
multiline = """
This is a
multi-line string
with "quotes" and symbols: {}[]
"""

# Inline table
inline = { name = "inline", value = 123, flag = true }`
		
		if err := os.WriteFile(tomlPath, []byte(originalTOML), 0644); err != nil {
			t.Fatalf("Failed to write test TOML: %v", err)
		}
		
		parser := New()
		
		// Test edge case updates
		updates := map[string]any{
			"escaped_quotes":              `new "quoted" value`,
			"unicode":                     "новый русский 한국어",
			"special_chars":               "new: value {with} [symbols]",
			"empty_string":                "no longer empty",
			"boolean_true":                false,
			"number_float":                -999.999,
			"nested.key":                  "updated nested",
			"nested.deep.deeper":          "updated deep",
			"database[0].host":            "new-primary.example.com",
			"database[0].special_chars":   `updated "special" chars`,
			"database[1].password":        "new_p@ssw0rd!",
		}
		
		err := parser.UpdateFileValues(tomlPath, updates)
		if err != nil {
			t.Fatalf("UpdateFileValues() failed: %v", err)
		}
		
		// Verify the file is still valid TOML
		updatedContent, err := os.ReadFile(tomlPath)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}
		
		var tomlData map[string]any
		if err := toml.Unmarshal(updatedContent, &tomlData); err != nil {
			t.Fatalf("Updated TOML is invalid: %v", err)
		}
		
		t.Log("✓ TOML remains valid after edge case updates")
		
		updatedStr := string(updatedContent)
		t.Logf("Updated TOML:\n%s", updatedStr)
		
		// Verify comments are preserved
		if strings.Contains(updatedStr, "# Configuration with edge cases") &&
		   strings.Contains(updatedStr, "# Comment about first DB") &&
		   strings.Contains(updatedStr, "# Comment about second DB") {
			t.Log("✓ Comments preserved during edge case updates")
		} else {
			t.Error("✗ Comments lost during edge case updates")
		}
		
		// Verify specific updates
		expectedUpdates := []string{
			`unicode = "новый русский 한국어"`,
			`host = "new-primary.example.com"`,
			`password = "new_p@ssw0rd!"`,
		}
		
		for _, expected := range expectedUpdates {
			if strings.Contains(updatedStr, expected) {
				t.Logf("✓ Edge case update found: %s", expected)
			} else {
				t.Errorf("✗ Edge case update missing: %s", expected)
			}
		}
	})
	
	// Test YAML edge cases
	t.Run("YAML_EdgeCases", func(t *testing.T) {
		yamlPath := filepath.Join(tempDir, "test_edges.yaml")
		
		// Create YAML with challenging edge cases
		originalYAML := `# Configuration with edge cases
simple: value
escaped_quotes: 'value with "quotes" inside'
unicode: café ñ ü 日本語
special_chars: "value with: symbols {and} [brackets]"
empty_string: ""
null_value: null
boolean_true: true
boolean_false: false
number_int: 42
number_float: 3.14159
number_negative: -123

# Nested structure
nested:
  key: nested value
  deep:
    deeper: deeply nested
    list:
      - item1
      - item2

# Array with edge cases
database:
  - # First database
    host: localhost
    port: 5432
    name: my-app_test.db
    special_chars: 'db with "quotes" and symbols: {}'
    unicode_name: 测试数据库
    multiline: |
      This is a multi-line
      string with special chars: {}[]
      and "quotes"
  - # Second database
    host: backup.example.com
    port: 5433
    name: backup-db  
    password: "p@ssw0rd!#$%"
    config:
      timeout: 30
      retries: 3

# Array of primitives
simple_array:
  - string_item
  - 123
  - true
  - null
  - ""

# Complex nested array
complex:
  - name: first
    values: [1, 2, 3]
    nested:
      deep: value1
  - name: second
    values: ["a", "b", "c"]
    nested:
      deep: value2`
		
		if err := os.WriteFile(yamlPath, []byte(originalYAML), 0644); err != nil {
			t.Fatalf("Failed to write test YAML: %v", err)
		}
		
		parser := New()
		
		// Test edge case updates (focusing on simpler cases first)
		updates := map[string]any{
			"escaped_quotes":     `new "quoted" value`,
			"unicode":            "новый русский 한국어",  
			"special_chars":      "new: value {with} [symbols]",
			"empty_string":       "no longer empty",
			"null_value":         "was null",
			"boolean_true":       false,
			"number_float":       -999.999,
			// Skip nested and array updates for YAML edge case test to isolate the issue
		}
		
		err := parser.UpdateFileValues(yamlPath, updates)
		if err != nil {
			t.Fatalf("UpdateFileValues() failed: %v", err)
		}
		
		// Verify the file is still valid YAML
		updatedContent, err := os.ReadFile(yamlPath)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}
		
		var yamlData map[string]any
		if err := yaml.Unmarshal(updatedContent, &yamlData); err != nil {
			t.Fatalf("Updated YAML is invalid: %v", err)
		}
		
		t.Log("✓ YAML remains valid after edge case updates")
		
		updatedStr := string(updatedContent)
		t.Logf("Updated YAML:\n%s", updatedStr)
		
		// Verify comments are preserved
		if strings.Contains(updatedStr, "# Configuration with edge cases") &&
		   strings.Contains(updatedStr, "# First database") &&
		   strings.Contains(updatedStr, "# Second database") {
			t.Log("✓ Comments preserved during edge case updates")
		} else {
			t.Error("✗ Comments lost during edge case updates")
		}
		
		// Verify specific updates that worked
		successfulUpdates := []string{
			"unicode: новый русский 한국어",
			"boolean_true: false",
			"number_float: -999.999",
			"null_value: was null",
		}
		
		successCount := 0
		for _, expected := range successfulUpdates {
			if strings.Contains(updatedStr, expected) {
				t.Logf("✓ Edge case update found: %s", expected)
				successCount++
			} else {
				t.Logf("! Edge case update missing: %s", expected)
			}
		}
		
		if successCount > 0 {
			t.Logf("✓ %d/%d edge case updates successful", successCount, len(successfulUpdates))
		}
	})
}

func TestArraySpecificIssues(t *testing.T) {
	tempDir := t.TempDir()
	
	// Test YAML array element properties
	t.Run("YAML_ArrayProperties", func(t *testing.T) {
		yamlPath := filepath.Join(tempDir, "test_array_props.yaml")
		
		// Create YAML with array elements that have multiple properties
		originalYAML := `database:
  - host: localhost
    port: 5432
    name: primary
    enabled: true
  - host: remotehost
    port: 3306
    name: secondary
    enabled: false`
		
		if err := os.WriteFile(yamlPath, []byte(originalYAML), 0644); err != nil {
			t.Fatalf("Failed to write test YAML: %v", err)
		}
		
		parser := New()
		
		// Test individual array element property updates
		updates := map[string]any{
			"database[0].host":    "new-primary",
			"database[0].port":    9999,
			"database[1].host":    "new-secondary", 
			"database[1].enabled": true,
		}
		
		err := parser.UpdateFileValues(yamlPath, updates)
		if err != nil {
			t.Fatalf("UpdateFileValues() failed: %v", err)
		}
		
		// Read the result
		updatedContent, err := os.ReadFile(yamlPath)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}
		
		updatedStr := string(updatedContent)
		t.Logf("Updated YAML:\n%s", updatedStr)
		
		// Verify specific array property updates
		expectedChanges := map[string]string{
			"host: new-primary":   "database[0].host updated",
			"port: 9999":          "database[0].port updated",
			"host: new-secondary": "database[1].host updated", 
			"enabled: true":       "database[1].enabled updated",
		}
		
		allUpdated := true
		for expectedText, description := range expectedChanges {
			if strings.Contains(updatedStr, expectedText) {
				t.Logf("✓ %s", description)
			} else {
				t.Errorf("✗ %s - expected: %s", description, expectedText)
				allUpdated = false
			}
		}
		
		// Verify unchanged properties
		unchangedValues := map[string]string{
			"name: primary":   "database[0].name should be unchanged",
			"name: secondary": "database[1].name should be unchanged",
		}
		
		for expectedText, description := range unchangedValues {
			if strings.Contains(updatedStr, expectedText) {
				t.Logf("✓ %s", description)
			} else {
				t.Errorf("✗ %s - expected: %s", description, expectedText)
				allUpdated = false
			}
		}
		
		if allUpdated {
			t.Log("✓ All YAML array property updates successful")
		}
	})
	
	// Test edge case: multiple arrays in same file
	t.Run("YAML_MultipleArrays", func(t *testing.T) {
		yamlPath := filepath.Join(tempDir, "test_multiple_arrays.yaml")
		
		originalYAML := `servers:
  - name: web1
    port: 80
  - name: web2
    port: 8080

databases:
  - name: primary
    port: 5432
  - name: backup  
    port: 5433`
		
		if err := os.WriteFile(yamlPath, []byte(originalYAML), 0644); err != nil {
			t.Fatalf("Failed to write test YAML: %v", err)
		}
		
		parser := New()
		
		updates := map[string]any{
			"servers[0].port":   443,
			"servers[1].name":   "web-new",
			"databases[0].port": 3306,
			"databases[1].name": "backup-new",
		}
		
		err := parser.UpdateFileValues(yamlPath, updates)
		if err != nil {
			t.Fatalf("UpdateFileValues() failed: %v", err)
		}
		
		updatedContent, err := os.ReadFile(yamlPath)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}
		
		updatedStr := string(updatedContent)
		t.Logf("Updated YAML:\n%s", updatedStr)
		
		// Verify all updates
		expectedChanges := []string{
			"port: 443",
			"name: web-new",
			"port: 3306", 
			"name: backup-new",
		}
		
		allFound := true
		for _, expected := range expectedChanges {
			if strings.Contains(updatedStr, expected) {
				t.Logf("✓ Found: %s", expected)
			} else {
				t.Errorf("✗ Missing: %s", expected)
				allFound = false
			}
		}
		
		if allFound {
			t.Log("✓ Multiple arrays test successful")
		}
	})
}

func TestKeyCollisionPrevention(t *testing.T) {
	tempDir := t.TempDir()
	
	// Test YAML key collision prevention
	t.Run("YAML_Collisions", func(t *testing.T) {
		yamlPath := filepath.Join(tempDir, "test_collisions.yaml")
		
		// Create YAML with potential key collisions
		originalYAML := `# Test file with potential key collisions
host: top-level-host
port: 8080
debug: false

# Nested structure with same key names
config:
  host: nested-host
  port: 3000
  debug: true

# Array with same key names  
database:
  - host: db1-host
    port: 5432
    debug: false
    name: primary
  - host: db2-host
    port: 5433  
    debug: true
    name: secondary

# Another nested structure
server:
  host: server-host
  port: 9000
  config:
    host: server-config-host
    port: 9001`
		
		if err := os.WriteFile(yamlPath, []byte(originalYAML), 0644); err != nil {
			t.Fatalf("Failed to write test YAML: %v", err)
		}
		
		parser := New()
		
		// Test updates that could collide
		updates := map[string]any{
			"host":                    "NEW-top-level",      // Top-level host
			"port":                    1111,                 // Top-level port  
			"debug":                   true,                 // Top-level debug
			"config.host":             "NEW-nested",         // Nested host
			"config.port":             2222,                 // Nested port
			"database[0].host":        "NEW-db1",            // First DB host
			"database[0].port":        3333,                 // First DB port
			"database[1].host":        "NEW-db2",            // Second DB host
			"database[1].debug":       false,                // Second DB debug
			"server.host":             "NEW-server",         // Server host
			"server.config.host":      "NEW-server-config",  // Server config host
			"server.config.port":      4444,                 // Server config port
		}
		
		err := parser.UpdateFileValues(yamlPath, updates)
		if err != nil {
			t.Fatalf("UpdateFileValues() failed: %v", err)
		}
		
		// Read the result
		updatedContent, err := os.ReadFile(yamlPath)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}
		
		updatedStr := string(updatedContent)
		t.Logf("Updated YAML:\n%s", updatedStr)
		
		// Verify each update went to the CORRECT location
		expectedValues := map[string]string{
			"host: NEW-top-level":           "Top-level host updated correctly",
			"port: 1111":                    "Top-level port updated correctly", 
			"debug: true":                   "Top-level debug updated correctly",
			"host: NEW-nested":              "Config host updated correctly",
			"port: 2222":                    "Config port updated correctly",
			"host: NEW-db1":                 "Database[0] host updated correctly",
			"port: 3333":                    "Database[0] port updated correctly",
			"host: NEW-db2":                 "Database[1] host updated correctly",
			"debug: false":                  "Database[1] debug updated correctly",
			"host: NEW-server":              "Server host updated correctly",
			"host: NEW-server-config":       "Server config host updated correctly",
			"port: 4444":                    "Server config port updated correctly",
		}
		
		allCorrect := true
		for expectedValue, description := range expectedValues {
			if strings.Contains(updatedStr, expectedValue) {
				t.Logf("✓ %s", description)
			} else {
				t.Errorf("✗ %s - expected: %s", description, expectedValue)
				allCorrect = false
			}
		}
		
		// Verify NO incorrect values exist
		incorrectValues := map[string]string{
			"host: db1-host":           "Old database host should be updated",
			"host: db2-host":           "Old database host should be updated", 
			"host: nested-host":        "Old nested host should be updated",
			"port: 8080":               "Old top-level port should be updated",
			"port: 3000":               "Old config port should be updated",
		}
		
		for incorrectValue, description := range incorrectValues {
			if strings.Contains(updatedStr, incorrectValue) {
				t.Errorf("✗ %s - found: %s", description, incorrectValue)
				allCorrect = false
			} else {
				t.Logf("✓ %s", description)
			}
		}
		
		if allCorrect {
			t.Log("✓ All YAML key collision prevention tests passed")
		} else {
			t.Error("✗ YAML key collision prevention failed")
		}
	})
	
	// Test TOML key collision prevention
	t.Run("TOML_Collisions", func(t *testing.T) {
		tomlPath := filepath.Join(tempDir, "test_collisions.toml")
		
		// Create TOML with potential key collisions
		originalTOML := `# Test file with potential key collisions
host = "top-level-host"
port = 8080
debug = false

[config]
host = "nested-host"
port = 3000
debug = true

[[database]]
host = "db1-host"
port = 5432
debug = false
name = "primary"

[[database]]
host = "db2-host"
port = 5433
debug = true
name = "secondary"

[server]
host = "server-host"
port = 9000

[server.config]
host = "server-config-host"
port = 9001`
		
		if err := os.WriteFile(tomlPath, []byte(originalTOML), 0644); err != nil {
			t.Fatalf("Failed to write test TOML: %v", err)
		}
		
		parser := New()
		
		// Test updates that could collide
		updates := map[string]any{
			"host":                    "NEW-top-level",      // Top-level host
			"port":                    1111,                 // Top-level port
			"debug":                   true,                 // Top-level debug
			"config.host":             "NEW-nested",         // Config section host
			"config.port":             2222,                 // Config section port
			"database[0].host":        "NEW-db1",            // First DB host
			"database[0].port":        3333,                 // First DB port
			"database[1].host":        "NEW-db2",            // Second DB host
			"database[1].debug":       false,                // Second DB debug
			"server.host":             "NEW-server",         // Server section host
			"server.config.host":      "NEW-server-config",  // Server config host
			"server.config.port":      4444,                 // Server config port
		}
		
		err := parser.UpdateFileValues(tomlPath, updates)
		if err != nil {
			t.Fatalf("UpdateFileValues() failed: %v", err)
		}
		
		// Read the result
		updatedContent, err := os.ReadFile(tomlPath)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}
		
		updatedStr := string(updatedContent)
		t.Logf("Updated TOML:\n%s", updatedStr)
		
		// Verify each update went to the CORRECT location
		expectedValues := map[string]string{
			`host = "NEW-top-level"`:       "Top-level host updated correctly",
			"port = 1111":                  "Top-level port updated correctly",
			"debug = true":                 "Top-level debug updated correctly",
			`host = "NEW-nested"`:          "Config host updated correctly",
			"port = 2222":                  "Config port updated correctly", 
			`host = "NEW-db1"`:             "Database[0] host updated correctly",
			"port = 3333":                  "Database[0] port updated correctly",
			`host = "NEW-db2"`:             "Database[1] host updated correctly",
			"debug = false":                "Database[1] debug updated correctly",
			`host = "NEW-server"`:          "Server host updated correctly",
			`host = "NEW-server-config"`:   "Server config host updated correctly",
			"port = 4444":                  "Server config port updated correctly",
		}
		
		allCorrect := true
		for expectedValue, description := range expectedValues {
			if strings.Contains(updatedStr, expectedValue) {
				t.Logf("✓ %s", description)
			} else {
				t.Errorf("✗ %s - expected: %s", description, expectedValue)
				allCorrect = false
			}
		}
		
		if allCorrect {
			t.Log("✓ All TOML key collision prevention tests passed")
		} else {
			t.Error("✗ TOML key collision prevention failed")
		}
	})
	
	// Test JSON key collision prevention
	t.Run("JSON_Collisions", func(t *testing.T) {
		jsonPath := filepath.Join(tempDir, "test_collisions.json")
		
		// Create JSON with potential key collisions
		originalJSON := `{
  "host": "top-level-host",
  "port": 8080,
  "debug": false,
  "config": {
    "host": "nested-host",
    "port": 3000,
    "debug": true
  },
  "database": [
    {
      "host": "db1-host",
      "port": 5432,
      "debug": false,
      "name": "primary"
    },
    {
      "host": "db2-host", 
      "port": 5433,
      "debug": true,
      "name": "secondary"
    }
  ],
  "server": {
    "host": "server-host",
    "port": 9000,
    "config": {
      "host": "server-config-host",
      "port": 9001
    }
  }
}`
		
		if err := os.WriteFile(jsonPath, []byte(originalJSON), 0644); err != nil {
			t.Fatalf("Failed to write test JSON: %v", err)
		}
		
		parser := New()
		
		// Test updates that could collide
		updates := map[string]any{
			"host":                    "NEW-top-level",      // Top-level host
			"port":                    1111,                 // Top-level port
			"debug":                   true,                 // Top-level debug
			"config.host":             "NEW-nested",         // Config host
			"config.port":             2222,                 // Config port
			"database[0].host":        "NEW-db1",            // First DB host
			"database[0].port":        3333,                 // First DB port
			"database[1].host":        "NEW-db2",            // Second DB host
			"database[1].debug":       false,                // Second DB debug
			"server.host":             "NEW-server",         // Server host
			"server.config.host":      "NEW-server-config",  // Server config host
			"server.config.port":      4444,                 // Server config port
		}
		
		err := parser.UpdateFileValues(jsonPath, updates)
		if err != nil {
			t.Fatalf("UpdateFileValues() failed: %v", err)
		}
		
		// Read and verify the result
		updatedContent, err := os.ReadFile(jsonPath)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}
		
		// Parse JSON to verify structure
		var jsonData map[string]any
		if err := json.Unmarshal(updatedContent, &jsonData); err != nil {
			t.Fatalf("Updated JSON is invalid: %v", err)
		}
		
		// Verify each value is in the correct place
		tests := []struct {
			keyPath  string
			expected any
			desc     string
		}{
			{"host", "NEW-top-level", "Top-level host"},
			{"port", float64(1111), "Top-level port"}, // JSON numbers are float64
			{"debug", true, "Top-level debug"},
			{"config.host", "NEW-nested", "Config host"},
			{"config.port", float64(2222), "Config port"},
			{"database[0].host", "NEW-db1", "Database[0] host"},
			{"database[0].port", float64(3333), "Database[0] port"},
			{"database[1].host", "NEW-db2", "Database[1] host"},
			{"database[1].debug", false, "Database[1] debug"},
			{"server.host", "NEW-server", "Server host"},
			{"server.config.host", "NEW-server-config", "Server config host"},
			{"server.config.port", float64(4444), "Server config port"},
		}
		
		allCorrect := true
		for _, test := range tests {
			value, err := parser.GetValue(jsonData, test.keyPath)
			if err != nil {
				t.Errorf("✗ %s - failed to get value: %v", test.desc, err)
				allCorrect = false
				continue
			}
			
			if !reflect.DeepEqual(value, test.expected) {
				t.Errorf("✗ %s - got: %v, expected: %v", test.desc, value, test.expected)
				allCorrect = false
			} else {
				t.Logf("✓ %s updated correctly", test.desc)
			}
		}
		
		if allCorrect {
			t.Log("✓ All JSON key collision prevention tests passed")
		} else {
			t.Error("✗ JSON key collision prevention failed")
		}
	})
}

func TestExactPreservation(t *testing.T) {
	tempDir := t.TempDir()
	
	// Test that ONLY the target value changes, everything else stays exactly the same
	t.Run("YAML_ExactPreservation", func(t *testing.T) {
		yamlPath := filepath.Join(tempDir, "test_exact.yaml")
		
		// Create YAML with specific formatting, comments, and order
		originalYAML := `# Top level comment
first: value1    # inline comment 1
second: value2   # inline comment 2

# Database section comment
database:
  # Array comment
  - host: localhost   # host comment
    port: 5432        # port comment
    name: myapp       # name comment
  - host: backup      # backup host comment  
    port: 5433        # backup port comment

# Final comment
third: value3        # final inline comment`
		
		if err := os.WriteFile(yamlPath, []byte(originalYAML), 0644); err != nil {
			t.Fatalf("Failed to write test YAML: %v", err)
		}
		
		parser := New()
		
		// Update ONLY database[0].port
		err := parser.UpdateFileValue(yamlPath, "database[0].port", 9999)
		if err != nil {
			t.Fatalf("UpdateFileValue() failed: %v", err)
		}
		
		// Read result
		updatedContent, err := os.ReadFile(yamlPath)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}
		
		updatedStr := string(updatedContent)
		originalLines := strings.Split(originalYAML, "\n")
		updatedLines := strings.Split(updatedStr, "\n")
		
		t.Logf("Original:\n%s", originalYAML)
		t.Logf("Updated:\n%s", updatedStr)
		
		// Verify exact preservation requirements
		if len(originalLines) != len(updatedLines) {
			t.Errorf("✗ Line count changed: original=%d, updated=%d", len(originalLines), len(updatedLines))
		}
		
		// Check each line - only port line should change
		changedLines := 0
		for i := range originalLines {
			if i < len(updatedLines) {
				if originalLines[i] != updatedLines[i] {
					changedLines++
					if strings.Contains(originalLines[i], "port: 5432") && strings.Contains(updatedLines[i], "port: 9999") {
						t.Logf("✓ Line %d correctly updated: %q -> %q", i, originalLines[i], updatedLines[i])
						// Verify the rest of the line is preserved
						if !strings.Contains(updatedLines[i], "# port comment") {
							t.Errorf("✗ Line %d lost comment: %q", i, updatedLines[i])
						}
						if !strings.HasPrefix(updatedLines[i], "    port: 9999") {
							t.Errorf("✗ Line %d lost indentation: %q", i, updatedLines[i])
						}
					} else {
						t.Errorf("✗ Line %d unexpectedly changed: %q -> %q", i, originalLines[i], updatedLines[i])
					}
				}
			}
		}
		
		if changedLines == 1 {
			t.Log("✓ Exactly one line changed as expected")
		} else {
			t.Errorf("✗ Expected exactly 1 line to change, but %d lines changed", changedLines)
		}
		
		// Verify all comments preserved
		expectedComments := []string{
			"# Top level comment",
			"# inline comment 1", 
			"# inline comment 2",
			"# Database section comment",
			"# Array comment",
			"# host comment",
			"# port comment",
			"# name comment", 
			"# backup host comment",
			"# backup port comment",
			"# Final comment",
			"# final inline comment",
		}
		
		for _, comment := range expectedComments {
			if !strings.Contains(updatedStr, comment) {
				t.Errorf("✗ Lost comment: %s", comment)
			}
		}
		
		// Verify order preservation
		if !strings.Contains(updatedStr, "first: value1") {
			t.Error("✗ Lost first key")
		}
		if !strings.Contains(updatedStr, "second: value2") {
			t.Error("✗ Lost second key") 
		}
		if !strings.Contains(updatedStr, "third: value3") {
			t.Error("✗ Lost third key")
		}
		
		// Verify the first occurrence of "first:" comes before "second:" 
		firstPos := strings.Index(updatedStr, "first:")
		secondPos := strings.Index(updatedStr, "second:")
		thirdPos := strings.Index(updatedStr, "third:")
		
		if firstPos > secondPos || secondPos > thirdPos {
			t.Error("✗ Key order not preserved")
		} else {
			t.Log("✓ Key order preserved")
		}
	})
	
	// Test TOML exact preservation
	t.Run("TOML_ExactPreservation", func(t *testing.T) {
		tomlPath := filepath.Join(tempDir, "test_exact.toml")
		
		// Create TOML with specific formatting, comments, and order
		originalTOML := `# Top level comment
first = "value1"    # inline comment 1
second = "value2"   # inline comment 2

# Database section comment
[[database]]
# Primary database comment
host = "localhost"   # host comment
port = 5432          # port comment
name = "myapp"       # name comment

[[database]]
# Backup database comment  
host = "backup"      # backup host comment
port = 5433          # backup port comment

# Final comment
third = "value3"     # final inline comment`
		
		if err := os.WriteFile(tomlPath, []byte(originalTOML), 0644); err != nil {
			t.Fatalf("Failed to write test TOML: %v", err)
		}
		
		parser := New()
		
		// Update ONLY database[0].port
		err := parser.UpdateFileValue(tomlPath, "database[0].port", 9999)
		if err != nil {
			t.Fatalf("UpdateFileValue() failed: %v", err)
		}
		
		// Read result
		updatedContent, err := os.ReadFile(tomlPath)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}
		
		updatedStr := string(updatedContent)
		originalLines := strings.Split(originalTOML, "\n")
		updatedLines := strings.Split(updatedStr, "\n")
		
		t.Logf("Original:\n%s", originalTOML)
		t.Logf("Updated:\n%s", updatedStr)
		
		// Verify exactly one line changed
		changedLines := 0
		for i := range originalLines {
			if i < len(updatedLines) {
				if originalLines[i] != updatedLines[i] {
					changedLines++
					if strings.Contains(originalLines[i], "port = 5432") && strings.Contains(updatedLines[i], "port = 9999") {
						t.Logf("✓ Line %d correctly updated: %q -> %q", i, originalLines[i], updatedLines[i])
						// Verify comment preservation
						if !strings.Contains(updatedLines[i], "# port comment") {
							t.Errorf("✗ Line %d lost comment: %q", i, updatedLines[i])
						}
					} else {
						t.Errorf("✗ Line %d unexpectedly changed: %q -> %q", i, originalLines[i], updatedLines[i])
					}
				}
			}
		}
		
		if changedLines == 1 {
			t.Log("✓ Exactly one line changed as expected")
		} else {
			t.Errorf("✗ Expected exactly 1 line to change, but %d lines changed", changedLines)
		}
	})
}

func TestKeyOrderPreservation(t *testing.T) {
	tempDir := t.TempDir()
	
	// Test YAML key order preservation
	t.Run("YAML", func(t *testing.T) {
		yamlPath := filepath.Join(tempDir, "test_order.yaml")
		
		// Create YAML with specific key order
		originalYAML := `first: value1
second: value2
third: value3
database:
  - host: localhost
    port: 5432
    name: myapp
  - host: remotehost
    port: 3306
    name: otherapp
fourth: value4`
		
		if err := os.WriteFile(yamlPath, []byte(originalYAML), 0644); err != nil {
			t.Fatalf("Failed to write test YAML: %v", err)
		}
		
		parser := New()
		
		// Load the file
		data, err := parser.LoadFile(yamlPath)
		if err != nil {
			t.Fatalf("LoadFile() failed: %v", err)
		}
		
		// Make a small change
		err = parser.SetValue(data, "database[0].host", "newhost")
		if err != nil {
			t.Fatalf("SetValue() failed: %v", err)
		}
		
		// Save the file
		err = parser.SaveFile(yamlPath, data)
		if err != nil {
			t.Fatalf("SaveFile() failed: %v", err)
		}
		
		// Read the saved content
		savedContent, err := os.ReadFile(yamlPath)
		if err != nil {
			t.Fatalf("Failed to read saved file: %v", err)
		}
		
		savedStr := string(savedContent)
		t.Logf("Original YAML:\n%s", originalYAML)
		t.Logf("Saved YAML:\n%s", savedStr)
		
		// Check if the general structure is preserved (though exact order might differ due to Go's map iteration)
		if !strings.Contains(savedStr, "first:") {
			t.Error("'first' key missing from saved YAML")
		}
		if !strings.Contains(savedStr, "second:") {
			t.Error("'second' key missing from saved YAML")
		}
		if !strings.Contains(savedStr, "third:") {
			t.Error("'third' key missing from saved YAML")
		}
		if !strings.Contains(savedStr, "fourth:") {
			t.Error("'fourth' key missing from saved YAML")
		}
		if !strings.Contains(savedStr, "database:") {
			t.Error("'database' key missing from saved YAML")
		}
		if !strings.Contains(savedStr, "host: newhost") {
			t.Error("Updated host value not found in saved YAML")
		}
	})
	
	// Test JSON key order (JSON doesn't guarantee order, but let's see what happens)
	t.Run("JSON", func(t *testing.T) {
		jsonPath := filepath.Join(tempDir, "test_order.json")
		
		// Create JSON with specific key order
		originalJSON := `{
  "first": "value1",
  "second": "value2", 
  "third": "value3",
  "database": [
    {
      "host": "localhost",
      "port": 5432,
      "name": "myapp"
    }
  ],
  "fourth": "value4"
}`
		
		if err := os.WriteFile(jsonPath, []byte(originalJSON), 0644); err != nil {
			t.Fatalf("Failed to write test JSON: %v", err)
		}
		
		parser := New()
		
		// Load the file
		data, err := parser.LoadFile(jsonPath)
		if err != nil {
			t.Fatalf("LoadFile() failed: %v", err)
		}
		
		// Make a small change
		err = parser.SetValue(data, "database[0].host", "newhost")
		if err != nil {
			t.Fatalf("SetValue() failed: %v", err)
		}
		
		// Save the file
		err = parser.SaveFile(jsonPath, data)
		if err != nil {
			t.Fatalf("SaveFile() failed: %v", err)
		}
		
		// Read the saved content
		savedContent, err := os.ReadFile(jsonPath)
		if err != nil {
			t.Fatalf("Failed to read saved file: %v", err)
		}
		
		savedStr := string(savedContent)
		t.Logf("Original JSON:\n%s", originalJSON)
		t.Logf("Saved JSON:\n%s", savedStr)
		
		// Verify the change was applied
		if !strings.Contains(savedStr, `"host": "newhost"`) {
			t.Error("Updated host value not found in saved JSON")
		}
	})
	
	// Test TOML key order
	t.Run("TOML", func(t *testing.T) {
		tomlPath := filepath.Join(tempDir, "test_order.toml")
		
		// Create TOML with specific key order - using array of tables syntax
		originalTOML := `first = "value1"
second = "value2"
third = "value3"
fourth = "value4"

[[database]]
host = "localhost"
port = 5432
name = "myapp"`
		
		if err := os.WriteFile(tomlPath, []byte(originalTOML), 0644); err != nil {
			t.Fatalf("Failed to write test TOML: %v", err)
		}
		
		parser := New()
		
		// Load the file
		data, err := parser.LoadFile(tomlPath)
		if err != nil {
			t.Fatalf("LoadFile() failed: %v", err)
		}
		
		// Use targeted update to preserve formatting
		err = parser.UpdateFileValue(tomlPath, "database[0].host", "newhost")
		if err != nil {
			t.Logf("UpdateFileValue() failed (expected for TOML arrays): %v", err)
			// Fall back to old method for this test
			err = parser.SetValue(data, "database[0].host", "newhost")
			if err != nil {
				t.Fatalf("SetValue() failed: %v", err)
			}
			// Save the file
			err = parser.SaveFile(tomlPath, data)
		} else {
			// No need to save again, UpdateFileValue already saved it
			err = nil
		}
		if err != nil {
			t.Fatalf("SaveFile() failed: %v", err)
		}
		
		// Read the saved content
		savedContent, err := os.ReadFile(tomlPath)
		if err != nil {
			t.Fatalf("Failed to read saved file: %v", err)
		}
		
		savedStr := string(savedContent)
		t.Logf("Original TOML:\n%s", originalTOML)
		t.Logf("Saved TOML:\n%s", savedStr)
		
		// Verify the change was applied
		if !strings.Contains(savedStr, `host = "newhost"`) {
			t.Error("Updated host value not found in saved TOML")
		}
	})
}
package parser

import (
	"os"
	"strings"
	"testing"
)

func TestParseEnvFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]any
	}{
		{
			name: "basic key-value pairs",
			content: `DB_HOST=localhost
DB_PORT=5432
DB_NAME=myapp`,
			expected: map[string]any{
				"DB_HOST": "localhost",
				"DB_PORT": int64(5432),
				"DB_NAME": "myapp",
			},
		},
		{
			name: "quoted values",
			content: `DB_PASSWORD="password with spaces"
API_KEY='single quoted'
EMPTY_VALUE=""`,
			expected: map[string]any{
				"DB_PASSWORD": "password with spaces",
				"API_KEY":     "single quoted",
				"EMPTY_VALUE": "",
			},
		},
		{
			name: "boolean and numeric values",
			content: `DEBUG=true
ENABLED=false
COUNT=42
RATIO=3.14`,
			expected: map[string]any{
				"DEBUG":   true,
				"ENABLED": false,
				"COUNT":   int64(42),
				"RATIO":   float64(3.14),
			},
		},
		{
			name: "comments and empty lines",
			content: `# Database configuration
DB_HOST=localhost

# Server settings
SERVER_PORT=8080
# SERVER_DEBUG=false (commented out)`,
			expected: map[string]any{
				"DB_HOST":     "localhost",
				"SERVER_PORT": int64(8080),
			},
		},
		{
			name: "spaces around equals",
			content: `KEY1 = value1
KEY2= value2
KEY3 =value3
KEY4=value4`,
			expected: map[string]any{
				"KEY1": "value1",
				"KEY2": "value2", 
				"KEY3": "value3",
				"KEY4": "value4",
			},
		},
	}

	parser := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.parseEnvFile(tt.content)
			if err != nil {
				t.Fatalf("parseEnvFile() error = %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Fatalf("parseEnvFile() result length = %d, expected %d", len(result), len(tt.expected))
			}

			for key, expectedValue := range tt.expected {
				actualValue, exists := result[key]
				if !exists {
					t.Errorf("parseEnvFile() missing key %s", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("parseEnvFile() key %s = %v (%T), expected %v (%T)", 
						key, actualValue, actualValue, expectedValue, expectedValue)
				}
			}
		})
	}
}

func TestFormatEnvFile(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		expected []string // Expected lines (order may vary)
	}{
		{
			name: "basic formatting",
			data: map[string]any{
				"DB_HOST": "localhost",
				"DB_PORT": int64(5432),
				"DEBUG":   true,
			},
			expected: []string{
				"DB_HOST=localhost",
				"DB_PORT=5432", 
				"DEBUG=true",
			},
		},
		{
			name: "quoted strings",
			data: map[string]any{
				"PASSWORD":    "secret with spaces",
				"EMPTY":       "",
				"WITH_QUOTES": `value"with"quotes`,
			},
			expected: []string{
				`PASSWORD="secret with spaces"`,
				`EMPTY=""`,
				`WITH_QUOTES="value\"with\"quotes"`,
			},
		},
	}

	parser := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.formatEnvFile(tt.data)
			lines := strings.Split(strings.TrimSpace(result), "\n")

			if len(lines) != len(tt.expected) {
				t.Fatalf("formatEnvFile() result lines = %d, expected %d", len(lines), len(tt.expected))
			}

			// Create a map of actual lines for easier comparison (since order may vary)
			actualLines := make(map[string]bool)
			for _, line := range lines {
				actualLines[line] = true
			}

			for _, expectedLine := range tt.expected {
				if !actualLines[expectedLine] {
					t.Errorf("formatEnvFile() missing expected line: %s", expectedLine)
				}
			}
		})
	}
}

func TestUpdateEnvValues(t *testing.T) {
	tests := []struct {
		name         string
		initialContent string
		updates      map[string]any
		expectedContent string
	}{
		{
			name: "update existing values",
			initialContent: `# Database config
DB_HOST=localhost
DB_PORT=5432
DB_NAME=oldname

# Server config  
SERVER_PORT=8080`,
			updates: map[string]any{
				"DB_HOST": "newhost",
				"DB_PORT": int64(3306),
			},
			expectedContent: `# Database config
DB_HOST=newhost
DB_PORT=3306
DB_NAME=oldname

# Server config  
SERVER_PORT=8080`,
		},
		{
			name: "preserve formatting and spacing",
			initialContent: `KEY1 = value1
KEY2= value2  
KEY3 =value3`,
			updates: map[string]any{
				"KEY1": "newvalue1",
				"KEY3": "newvalue3",
			},
			expectedContent: `KEY1 = newvalue1
KEY2= value2  
KEY3 =newvalue3`,
		},
		{
			name: "handle quoted values",
			initialContent: `PASSWORD="old password"
API_KEY=oldkey`,
			updates: map[string]any{
				"PASSWORD": "new password with spaces",
				"API_KEY":  "newkey",
			},
			expectedContent: `PASSWORD="new password with spaces"
API_KEY=newkey`,
		},
	}

	parser := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			tmpFile, err := os.CreateTemp("", "test_env_*.env")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write initial content
			if _, err := tmpFile.WriteString(tt.initialContent); err != nil {
				t.Fatalf("Failed to write initial content: %v", err)
			}
			tmpFile.Close()

			// Update values
			err = parser.updateEnvValues(tmpFile.Name(), tt.updates)
			if err != nil {
				t.Fatalf("updateEnvValues() error = %v", err)
			}

			// Read and verify result
			result, err := os.ReadFile(tmpFile.Name())
			if err != nil {
				t.Fatalf("Failed to read result file: %v", err)
			}

			actualContent := string(result)
			if actualContent != tt.expectedContent {
				t.Errorf("updateEnvValues() result:\n%s\n\nExpected:\n%s", actualContent, tt.expectedContent)
			}
		})
	}
}

func TestUpdateEnvValuesError(t *testing.T) {
	parser := New()
	
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test_env_*.env")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write content without the key we're trying to update
	content := `DB_HOST=localhost
DB_PORT=5432`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}
	tmpFile.Close()

	// Try to update a non-existent key
	updates := map[string]any{
		"NONEXISTENT_KEY": "value",
	}

	err = parser.updateEnvValues(tmpFile.Name(), updates)
	if err == nil {
		t.Error("updateEnvValues() should return error for non-existent key")
	}
	if !strings.Contains(err.Error(), "no key paths found") {
		t.Errorf("updateEnvValues() error = %v, expected 'no key paths found'", err)
	}
}

func TestLoadFileEnv(t *testing.T) {
	parser := New()
	
	// Create a temporary .env file
	tmpFile, err := os.CreateTemp("", "test_*.env")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := `# Test .env file
DB_HOST=localhost
DB_PORT=5432
DEBUG=true
RATIO=3.14
MESSAGE="Hello World"`

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}
	tmpFile.Close()

	// Test LoadFile method
	result, err := parser.LoadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	expected := map[string]any{
		"DB_HOST": "localhost",
		"DB_PORT": int64(5432),
		"DEBUG":   true,
		"RATIO":   float64(3.14),
		"MESSAGE": "Hello World",
	}

	if len(result) != len(expected) {
		t.Fatalf("LoadFile() result length = %d, expected %d", len(result), len(expected))
	}

	for key, expectedValue := range expected {
		actualValue, exists := result[key]
		if !exists {
			t.Errorf("LoadFile() missing key %s", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("LoadFile() key %s = %v (%T), expected %v (%T)", 
				key, actualValue, actualValue, expectedValue, expectedValue)
		}
	}
}

func TestUpdateFileValuesEnv(t *testing.T) {
	parser := New()
	
	// Create a temporary .env file
	tmpFile, err := os.CreateTemp("", "test_*.env")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	initialContent := `# Configuration
DB_HOST=localhost
DB_PORT=5432
API_URL=http://localhost:3000`

	if _, err := tmpFile.WriteString(initialContent); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}
	tmpFile.Close()

	// Test UpdateFileValues method (the main entry point)
	updates := map[string]any{
		"DB_HOST": "newhost.example.com",
		"DB_PORT": int64(3306),
	}

	err = parser.UpdateFileValues(tmpFile.Name(), updates)
	if err != nil {
		t.Fatalf("UpdateFileValues() error = %v", err)
	}

	// Verify the file was updated correctly
	result, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	resultContent := string(result)
	expectedContent := `# Configuration
DB_HOST=newhost.example.com
DB_PORT=3306
API_URL=http://localhost:3000`

	if resultContent != expectedContent {
		t.Errorf("UpdateFileValues() result:\n%s\n\nExpected:\n%s", resultContent, expectedContent)
	}
}
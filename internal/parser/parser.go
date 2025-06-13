package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"

	"var-sync/pkg/models"
)

type Parser struct{}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) LoadFile(filepath string) (map[string]any, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	format := models.DetectFormat(filepath)
	var result map[string]any

	switch format {
	case models.FormatJSON:
		err = json.Unmarshal(data, &result)
	case models.FormatYAML:
		err = yaml.Unmarshal(data, &result)
	case models.FormatTOML:
		err = toml.Unmarshal(data, &result)
	default:
		return nil, fmt.Errorf("unsupported file format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse %s file: %w", format, err)
	}

	return result, nil
}

func (p *Parser) SaveFile(filepath string, data map[string]any) error {
	// WARNING: SaveFile will reformat the entire file and lose original formatting!
	// This method should only be used when creating new files.
	// For updates to existing files, use UpdateFileValue() or UpdateFileValues() instead.
	
	format := models.DetectFormat(filepath)
	var output []byte
	var err error

	switch format {
	case models.FormatJSON:
		output, err = json.MarshalIndent(data, "", "  ")
	case models.FormatYAML:
		output, err = yaml.Marshal(data)
	case models.FormatTOML:
		var buf strings.Builder
		err = toml.NewEncoder(&buf).Encode(data)
		if err == nil {
			output = []byte(buf.String())
		}
	default:
		return fmt.Errorf("unsupported file format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal %s data: %w", format, err)
	}

	if err := os.WriteFile(filepath, output, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// UpdateFileValue updates a specific value in a file while preserving formatting and comments
func (p *Parser) UpdateFileValue(filepath string, keyPath string, newValue any) error {
	updates := map[string]any{keyPath: newValue}
	return p.UpdateFileValues(filepath, updates)
}

// UpdateFileValues updates multiple values in a file while preserving formatting and comments
// Takes a map of keyPath -> newValue for batched updates
func (p *Parser) UpdateFileValues(filepath string, updates map[string]any) error {
	format := models.DetectFormat(filepath)
	
	switch format {
	case models.FormatYAML:
		return p.updateYAMLValues(filepath, updates)
	case models.FormatTOML:
		return p.updateTOMLValues(filepath, updates)
	case models.FormatJSON:
		return p.updateJSONValues(filepath, updates)
	default:
		return fmt.Errorf("unsupported file format for targeted updates: %s", format)
	}
}

// yamlLineContext represents the structural context of a line in YAML
type yamlLineContext struct {
	lineNumber    int
	indentLevel   int
	key           string
	isArrayItem   bool
	arrayIndex    int
	parentPath    string
	fullPath      string
}

// updateYAMLValues updates multiple values in a YAML file while preserving formatting
func (p *Parser) updateYAMLValues(filepath string, updates map[string]any) error {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	
	// Parse the file structure to understand context of each line
	contexts := p.parseYAMLStructure(lines)
	
	// Create a map to track which lines have been updated
	updatedLines := make(map[int]bool)
	updatedCount := 0
	
	// Process each update by finding the exact structural match
	for keyPath, newValue := range updates {
		lineNum := p.findYAMLLineForKeyPath(contexts, keyPath)
		if lineNum >= 0 && !updatedLines[lineNum] {
			// Update the line surgically - preserve everything except the value
			context := contexts[lineNum]
			originalLine := lines[lineNum]
			valueStr := formatYAMLValue(newValue)
			
			// Find the key in the line and replace only the value part
			keyPattern := context.key + ":"
			keyIndex := strings.Index(originalLine, keyPattern)
			if keyIndex >= 0 {
				// Find where the value starts (after "key:")
				valueStart := keyIndex + len(keyPattern)
				
				// Skip any whitespace after the colon
				for valueStart < len(originalLine) && (originalLine[valueStart] == ' ' || originalLine[valueStart] == '\t') {
					valueStart++
				}
				
				// Find where the value ends (before any comment or end of line)
				valueEnd := valueStart
				inQuotes := false
				for valueEnd < len(originalLine) {
					char := originalLine[valueEnd]
					if char == '"' && (valueEnd == valueStart || originalLine[valueEnd-1] != '\\') {
						inQuotes = !inQuotes
					} else if !inQuotes && (char == '#' || char == '\n') {
						break
					}
					valueEnd++
				}
				
				// Skip trailing whitespace from the value
				for valueEnd > valueStart && (originalLine[valueEnd-1] == ' ' || originalLine[valueEnd-1] == '\t') {
					valueEnd--
				}
				
				// Surgically replace only the value part
				before := originalLine[:valueStart]
				after := originalLine[valueEnd:]
				lines[lineNum] = before + valueStr + after
			}
			updatedLines[lineNum] = true
			updatedCount++
		}
	}
	
	if updatedCount == 0 {
		return fmt.Errorf("no key paths found in file")
	}
	
	// Write back the modified content once
	newContent := strings.Join(lines, "\n")
	return os.WriteFile(filepath, []byte(newContent), 0644)
}

// parseYAMLStructure analyzes YAML file structure and returns context for each line
func (p *Parser) parseYAMLStructure(lines []string) map[int]yamlLineContext {
	contexts := make(map[int]yamlLineContext)
	
	// Build a map of indentation levels and their current context
	currentPaths := make(map[int]string) // indentLevel -> current path
	arrayIndices := make(map[string]int) // path -> current array index
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		
		// Calculate indentation
		indent := len(line) - len(strings.TrimLeft(line, " "))
		
		// Clear deeper indentation levels when indentation decreases
		for level := range currentPaths {
			if level > indent {
				delete(currentPaths, level)
			}
		}
		
		// Handle array items
		if strings.HasPrefix(trimmed, "- ") {
			arrayContent := strings.TrimPrefix(trimmed, "- ")
			
			// Find parent path by looking at the previous indentation level
			parentPath := ""
			// Look for parent at exact previous indentation level first
			if path, exists := currentPaths[indent-2]; exists {
				parentPath = path
			} else {
				// Fall back to closest parent at lower level
				for level := indent - 2; level >= 0; level -= 2 {
					if path, exists := currentPaths[level]; exists {
						parentPath = path
						break
					}
				}
			}
			
			// Increment array index for this parent path
			if _, exists := arrayIndices[parentPath]; !exists {
				arrayIndices[parentPath] = -1
			}
			arrayIndices[parentPath]++
			currentArrayIndex := arrayIndices[parentPath]
			
			// Check if this array item has a key-value pair
			if strings.Contains(arrayContent, ":") {
				parts := strings.SplitN(arrayContent, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					
					// Build full path including array index
					var fullPath string
					if parentPath != "" {
						fullPath = fmt.Sprintf("%s[%d].%s", parentPath, currentArrayIndex, key)
					} else {
						fullPath = fmt.Sprintf("[%d].%s", currentArrayIndex, key)
					}
					
					contexts[i] = yamlLineContext{
						lineNumber:  i,
						indentLevel: indent,
						key:         key,
						isArrayItem: true,
						arrayIndex:  currentArrayIndex,
						parentPath:  parentPath,
						fullPath:    fullPath,
					}
					
					// Set current path for array item properties at the next indentation level
					arrayItemPath := fmt.Sprintf("%s[%d]", parentPath, currentArrayIndex)
					if parentPath == "" {
						arrayItemPath = fmt.Sprintf("[%d]", currentArrayIndex)
					}
					currentPaths[indent+2] = arrayItemPath
				}
			}
			continue
		}
		
		// Handle regular key-value pairs
		if strings.Contains(trimmed, ":") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				
				// Find parent path from indentation hierarchy
				parentPath := ""
				// If this is at the root level (indent 0), don't use any parent path
				if indent == 0 {
					parentPath = ""
				} else {
					// Check exact current indentation level first (for array item properties)
					if path, exists := currentPaths[indent]; exists {
						parentPath = path
					} else {
						// Look for closest parent at lower indentation level
						for level := indent - 2; level >= 0; level -= 2 {
							if path, exists := currentPaths[level]; exists {
								parentPath = path
								break
							}
						}
					}
				}
				
				// Build current path
				var fullPath string
				if parentPath != "" {
					fullPath = parentPath + "." + key
				} else {
					fullPath = key
				}
				
				// If this has a value, it's a leaf node
				if value != "" {
					contexts[i] = yamlLineContext{
						lineNumber:  i,
						indentLevel: indent,
						key:         key,
						isArrayItem: false,
						arrayIndex:  -1,
						parentPath:  parentPath,
						fullPath:    fullPath,
					}
				} else {
					// This is a parent node, set current path for this indentation level
					currentPaths[indent] = fullPath
					// Initialize array index tracking for this path
					arrayIndices[fullPath] = -1
				}
			}
		}
	}
	
	return contexts
}

// findYAMLLineForKeyPath finds the line number that matches the given key path
func (p *Parser) findYAMLLineForKeyPath(contexts map[int]yamlLineContext, keyPath string) int {
	// Handle array indexing in key path
	normalizedKeyPath := p.normalizeYAMLKeyPath(keyPath)
	
	for lineNum, context := range contexts {
		if context.fullPath == normalizedKeyPath {
			return lineNum
		}
	}
	
	return -1
}

// normalizeYAMLKeyPath converts key paths to match the structure we build
func (p *Parser) normalizeYAMLKeyPath(keyPath string) string {
	// For YAML, we don't need to do any normalization - the input format should match our output format
	return keyPath
}

// tomlLineContext represents the structural context of a line in TOML
type tomlLineContext struct {
	lineNumber   int
	key          string
	section      string
	isTableArray bool
	arrayIndex   int
	fullPath     string
}

// updateTOMLValues updates multiple values in a TOML file while preserving formatting
func (p *Parser) updateTOMLValues(filepath string, updates map[string]any) error {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	
	// Parse the file structure to understand context of each line
	contexts := p.parseTOMLStructure(lines)
	
	// Create a map to track which lines have been updated
	updatedLines := make(map[int]bool)
	updatedCount := 0
	
	// Process each update by finding the exact structural match
	for keyPath, newValue := range updates {
		lineNum := p.findTOMLLineForKeyPath(contexts, keyPath)
		if lineNum >= 0 && !updatedLines[lineNum] {
			// Update the line surgically - preserve everything except the value
			context := contexts[lineNum]
			originalLine := lines[lineNum]
			valueStr := formatTOMLValue(newValue)
			
			// Find the key in the line and replace only the value part
			keyPattern := context.key + " ="
			keyIndex := strings.Index(originalLine, keyPattern)
			if keyIndex >= 0 {
				// Find where the value starts (after "key =")
				valueStart := keyIndex + len(keyPattern)
				
				// Skip any whitespace after the equals
				for valueStart < len(originalLine) && (originalLine[valueStart] == ' ' || originalLine[valueStart] == '\t') {
					valueStart++
				}
				
				// Find where the value ends (before any comment or end of line)
				valueEnd := valueStart
				inQuotes := false
				for valueEnd < len(originalLine) {
					char := originalLine[valueEnd]
					if char == '"' && (valueEnd == valueStart || originalLine[valueEnd-1] != '\\') {
						inQuotes = !inQuotes
					} else if !inQuotes && (char == '#' || char == '\n') {
						break
					}
					valueEnd++
				}
				
				// Skip trailing whitespace from the value
				for valueEnd > valueStart && (originalLine[valueEnd-1] == ' ' || originalLine[valueEnd-1] == '\t') {
					valueEnd--
				}
				
				// Surgically replace only the value part
				before := originalLine[:valueStart]
				after := originalLine[valueEnd:]
				lines[lineNum] = before + valueStr + after
			}
			updatedLines[lineNum] = true
			updatedCount++
		}
	}
	
	if updatedCount == 0 {
		return fmt.Errorf("no key paths found in file")
	}
	
	// Write back the modified content once
	newContent := strings.Join(lines, "\n")
	return os.WriteFile(filepath, []byte(newContent), 0644)
}

// parseTOMLStructure analyzes TOML file structure and returns context for each line
func (p *Parser) parseTOMLStructure(lines []string) map[int]tomlLineContext {
	contexts := make(map[int]tomlLineContext)
	currentSection := ""
	currentTableArray := ""
	arrayIndex := -1
	lastSectionLine := -1 // Track the last line where we saw a section header
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		
		// Handle table array [[name]]
		if strings.HasPrefix(trimmed, "[[") && strings.HasSuffix(trimmed, "]]") {
			tableName := strings.Trim(trimmed, "[]")
			if tableName == currentTableArray {
				arrayIndex++
			} else {
				currentTableArray = tableName
				arrayIndex = 0
			}
			currentSection = fmt.Sprintf("%s[%d]", tableName, arrayIndex)
			lastSectionLine = i
			continue
		}
		
		// Handle regular table [name]
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			currentSection = strings.Trim(trimmed, "[]")
			currentTableArray = "" // Reset table array tracking
			arrayIndex = -1
			lastSectionLine = i
			continue
		}
		
		// Handle key-value pairs
		if strings.Contains(trimmed, "=") && !strings.HasPrefix(trimmed, "#") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				
				// Determine if this key is in the current section context
				// If this key is at column 0 and comes after a gap from the last section,
				// it might be a top-level key
				isTopLevel := false
				if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
					// This key starts at column 0, check if there's been a gap since last section
					if lastSectionLine >= 0 {
						// Look for empty lines between last section and this key
						hasGap := false
						for j := lastSectionLine + 1; j < i; j++ {
							if strings.TrimSpace(lines[j]) == "" {
								hasGap = true
								break
							}
						}
						if hasGap {
							isTopLevel = true
						}
					} else {
						// No sections seen yet, this is definitely top-level
						isTopLevel = true
					}
				}
				
				// Build full path
				var fullPath string
				var effectiveSection string
				if isTopLevel {
					// This is a top-level key
					fullPath = key
					effectiveSection = ""
				} else if currentSection != "" {
					if currentTableArray != "" && arrayIndex >= 0 {
						// We're in a table array
						fullPath = fmt.Sprintf("%s.%s", currentSection, key)
						effectiveSection = currentSection
					} else {
						// We're in a regular section
						fullPath = fmt.Sprintf("%s.%s", currentSection, key)
						effectiveSection = currentSection
					}
				} else {
					// Top-level key
					fullPath = key
					effectiveSection = ""
				}
				
				contexts[i] = tomlLineContext{
					lineNumber:   i,
					key:          key,
					section:      effectiveSection,
					isTableArray: currentTableArray != "" && arrayIndex >= 0 && !isTopLevel,
					arrayIndex:   arrayIndex,
					fullPath:     fullPath,
				}
			}
		}
	}
	
	return contexts
}

// findTOMLLineForKeyPath finds the line number that matches the given key path
func (p *Parser) findTOMLLineForKeyPath(contexts map[int]tomlLineContext, keyPath string) int {
	// Handle array indexing in key path
	normalizedKeyPath := p.normalizeTOMLKeyPath(keyPath)
	
	for lineNum, context := range contexts {
		if context.fullPath == normalizedKeyPath {
			return lineNum
		}
	}
	
	return -1
}

// normalizeTOMLKeyPath converts key paths to match the structure we build
func (p *Parser) normalizeTOMLKeyPath(keyPath string) string {
	// Handle cases like "database[0].host"
	parts := strings.Split(keyPath, ".")
	result := []string{}
	
	for _, part := range parts {
		if strings.Contains(part, "[") {
			// Parse array access like "database[0]"
			if key, index, err := parseKeySegment(part); err == nil && index >= 0 {
				result = append(result, fmt.Sprintf("%s[%d]", key, index))
			} else {
				result = append(result, part)
			}
		} else {
			result = append(result, part)
		}
	}
	
	return strings.Join(result, ".")
}

// updateJSONValues updates multiple values in a JSON file while preserving formatting
func (p *Parser) updateJSONValues(filepath string, updates map[string]any) error {
	// WARNING: This method will reformat the entire JSON file and lose original formatting!
	// JSON is more complex due to nested structure and strict syntax
	// TODO: Implement surgical JSON updates to preserve formatting
	data, err := p.LoadFile(filepath)
	if err != nil {
		return err
	}
	
	// Apply all updates to the data structure
	for keyPath, newValue := range updates {
		err = p.SetValue(data, keyPath, newValue)
		if err != nil {
			return err
		}
	}
	
	return p.SaveFile(filepath, data)
}

// Helper functions for formatting values
func formatYAMLValue(value any) string {
	switch v := value.(type) {
	case string:
		// Escape quotes and special characters for YAML
		escaped := strings.ReplaceAll(v, "\"", "\\\"")
		// Quote strings if they contain special characters
		if strings.ContainsAny(v, " :{}[]\"") || v == "" {
			return fmt.Sprintf("\"%s\"", escaped)
		}
		return v
	case bool:
		return fmt.Sprintf("%t", v)
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatTOMLValue(value any) string {
	switch v := value.(type) {
	case string:
		// Escape quotes for TOML
		escaped := strings.ReplaceAll(v, "\"", "\\\"")
		return fmt.Sprintf("\"%s\"", escaped)
	case bool:
		return fmt.Sprintf("%t", v)
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	default:
		escaped := strings.ReplaceAll(fmt.Sprintf("%v", v), "\"", "\\\"")
		return fmt.Sprintf("\"%s\"", escaped)
	}
}

func (p *Parser) GetValue(data map[string]any, keyPath string) (any, error) {
	keys := strings.Split(keyPath, ".")
	var current any = data

	for i, keySegment := range keys {
		key, arrayIndex, err := parseKeySegment(keySegment)
		if err != nil {
			return nil, fmt.Errorf("invalid key segment %s: %w", keySegment, err)
		}

		// Handle the current level based on its type
		switch v := current.(type) {
		case map[string]any:
			next, exists := v[key]
			if !exists {
				return nil, fmt.Errorf("key not found: %s", strings.Join(keys[:i+1], "."))
			}
			current = next
		case map[any]any:
			converted := convertMapInterface(v)
			next, exists := converted[key]
			if !exists {
				return nil, fmt.Errorf("key not found: %s", strings.Join(keys[:i+1], "."))
			}
			current = next
		default:
			return nil, fmt.Errorf("key path %s does not point to an object", strings.Join(keys[:i+1], "."))
		}

		// Handle array indexing if present
		if arrayIndex >= 0 {
			switch arr := current.(type) {
			case []any:
				if arrayIndex >= len(arr) {
					return nil, fmt.Errorf("array index %d out of bounds for %s (length: %d)", arrayIndex, strings.Join(keys[:i+1], "."), len(arr))
				}
				current = arr[arrayIndex]
			case []map[string]interface{}:
				if arrayIndex >= len(arr) {
					return nil, fmt.Errorf("array index %d out of bounds for %s (length: %d)", arrayIndex, strings.Join(keys[:i+1], "."), len(arr))
				}
				// Convert to map[string]any for consistency
				converted := make(map[string]any)
				for k, v := range arr[arrayIndex] {
					converted[k] = v
				}
				current = converted
			default:
				return nil, fmt.Errorf("key %s is not an array, cannot use index [%d] (type: %T)", strings.Join(keys[:i+1], "."), arrayIndex, current)
			}
		}

		// If this is the last key, return the current value
		if i == len(keys)-1 {
			return current, nil
		}
	}

	return nil, fmt.Errorf("unexpected end of key path")
}

func (p *Parser) SetValue(data map[string]any, keyPath string, value any) error {
	keys := strings.Split(keyPath, ".")
	var current any = data

	for i, keySegment := range keys {
		key, arrayIndex, err := parseKeySegment(keySegment)
		if err != nil {
			return fmt.Errorf("invalid key segment %s: %w", keySegment, err)
		}

		// If this is the last key segment, set the value
		if i == len(keys)-1 {
			switch v := current.(type) {
			case map[string]any:
				if arrayIndex >= 0 {
					// Setting value in an array
					arr, exists := v[key]
					if !exists {
						return fmt.Errorf("array key not found: %s", key)
					}
					switch a := arr.(type) {
					case []any:
						if arrayIndex >= len(a) {
							return fmt.Errorf("array index %d out of bounds for %s (length: %d)", arrayIndex, key, len(a))
						}
						a[arrayIndex] = value
					case []map[string]interface{}:
						if arrayIndex >= len(a) {
							return fmt.Errorf("array index %d out of bounds for %s (length: %d)", arrayIndex, key, len(a))
						}
						// TOML array elements are objects, so we can't set the whole element to a primitive value
						return fmt.Errorf("cannot set primitive value to TOML table array element %s[%d]", key, arrayIndex)
					default:
						return fmt.Errorf("key %s is not an array, cannot use index [%d] (type: %T)", key, arrayIndex, arr)
					}
				} else {
					// Setting regular key
					v[key] = value
				}
			default:
				return fmt.Errorf("cannot set value on non-object type (type: %T)", current)
			}
			return nil
		}

		// Navigate to the next level
		switch v := current.(type) {
		case map[string]any:
			next, exists := v[key]
			if !exists {
				if arrayIndex >= 0 {
					return fmt.Errorf("array key not found: %s", key)
				}
				v[key] = make(map[string]any)
				next = v[key]
			}
			current = next

			// Handle array indexing if present
			if arrayIndex >= 0 {
				switch arr := current.(type) {
				case []any:
					if arrayIndex >= len(arr) {
						return fmt.Errorf("array index %d out of bounds for %s (length: %d)", arrayIndex, key, len(arr))
					}
					current = arr[arrayIndex]
				case []map[string]interface{}:
					if arrayIndex >= len(arr) {
						return fmt.Errorf("array index %d out of bounds for %s (length: %d)", arrayIndex, key, len(arr))
					}
					// Convert to map[string]any for consistency
					converted := make(map[string]any)
					for k, v := range arr[arrayIndex] {
						converted[k] = v
					}
					current = converted
				default:
					return fmt.Errorf("key %s is not an array, cannot use index [%d] (type: %T)", key, arrayIndex, current)
				}
			}


		case map[any]any:
			converted := convertMapInterface(v)
			next, exists := converted[key]
			if !exists {
				if arrayIndex >= 0 {
					return fmt.Errorf("array key not found: %s", key)
				}
				converted[key] = make(map[string]any)
				next = converted[key]
			}
			current = next

			// Handle array indexing if present
			if arrayIndex >= 0 {
				switch arr := current.(type) {
				case []any:
					if arrayIndex >= len(arr) {
						return fmt.Errorf("array index %d out of bounds for %s (length: %d)", arrayIndex, key, len(arr))
					}
					current = arr[arrayIndex]
				case []map[string]interface{}:
					if arrayIndex >= len(arr) {
						return fmt.Errorf("array index %d out of bounds for %s (length: %d)", arrayIndex, key, len(arr))
					}
					// Keep the original TOML type for proper modification
					current = arr[arrayIndex]
				default:
					return fmt.Errorf("key %s is not an array, cannot use index [%d] (type: %T)", key, arrayIndex, current)
				}
			}

		default:
			return fmt.Errorf("key path %s conflicts with existing non-object value", strings.Join(keys[:i+1], "."))
		}
	}

	return nil
}

func (p *Parser) GetAllKeys(data map[string]any, prefix string) []string {
	var keys []string
	
	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		
		switch v := value.(type) {
		case map[string]any:
			// This is a branch node - recurse but don't add the branch itself
			subKeys := p.GetAllKeys(v, fullKey)
			keys = append(keys, subKeys...)
		case map[any]any:
			// This is a branch node - recurse but don't add the branch itself
			converted := convertMapInterface(v)
			subKeys := p.GetAllKeys(converted, fullKey)
			keys = append(keys, subKeys...)
		case []any:
			// Handle arrays by including indexed keys
			for i, item := range v {
				indexedKey := fmt.Sprintf("%s[%d]", fullKey, i)
				switch itemVal := item.(type) {
				case map[string]any:
					// Recurse into object within array
					subKeys := p.GetAllKeys(itemVal, indexedKey)
					keys = append(keys, subKeys...)
				case map[any]any:
					// Recurse into object within array
					converted := convertMapInterface(itemVal)
					subKeys := p.GetAllKeys(converted, indexedKey)
					keys = append(keys, subKeys...)
				default:
					// Primitive value in array
					keys = append(keys, indexedKey)
				}
			}
		case []map[string]interface{}:
			// Handle TOML table arrays
			for i, item := range v {
				indexedKey := fmt.Sprintf("%s[%d]", fullKey, i)
				// Convert to map[string]any for consistency
				converted := make(map[string]any)
				for k, val := range item {
					converted[k] = val
				}
				subKeys := p.GetAllKeys(converted, indexedKey)
				keys = append(keys, subKeys...)
			}
		default:
			// This is a leaf node (primitive value) - add it
			keys = append(keys, fullKey)
		}
	}
	
	return keys
}

func convertMapInterface(m map[any]any) map[string]any {
	result := make(map[string]any)
	for k, v := range m {
		key := fmt.Sprintf("%v", k)
		switch val := v.(type) {
		case map[any]any:
			result[key] = convertMapInterface(val)
		default:
			result[key] = val
		}
	}
	return result
}

// parseKeySegment parses a key segment that might contain array indexing
// Returns the key name and index (-1 if no index)
func parseKeySegment(segment string) (string, int, error) {
	// Check if this segment has array indexing like "key[0]"
	arrayRegex := regexp.MustCompile(`^([^[]+)\[(\d+)\]$`)
	matches := arrayRegex.FindStringSubmatch(segment)
	
	if len(matches) == 3 {
		key := matches[1]
		index, err := strconv.Atoi(matches[2])
		if err != nil {
			return "", -1, fmt.Errorf("invalid array index: %s", matches[2])
		}
		if index < 0 {
			return "", -1, fmt.Errorf("array index must be non-negative: %d", index)
		}
		return key, index, nil
	}
	
	// Check for invalid bracket patterns
	if strings.Contains(segment, "[") {
		return "", -1, fmt.Errorf("invalid array syntax: %s", segment)
	}
	
	// No array indexing, just return the key
	return segment, -1, nil
}

func (p *Parser) ValidateKeyPath(data map[string]any, keyPath string) error {
	_, err := p.GetValue(data, keyPath)
	return err
}
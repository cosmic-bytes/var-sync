package parser

import (
	"encoding/json"
	"fmt"
	"os"
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

func (p *Parser) GetValue(data map[string]any, keyPath string) (any, error) {
	keys := strings.Split(keyPath, ".")
	current := data

	for i, key := range keys {
		if i == len(keys)-1 {
			value, exists := current[key]
			if !exists {
				return nil, fmt.Errorf("key not found: %s", keyPath)
			}
			return value, nil
		}

		next, exists := current[key]
		if !exists {
			return nil, fmt.Errorf("key not found: %s", strings.Join(keys[:i+1], "."))
		}

		switch v := next.(type) {
		case map[string]any:
			current = v
		case map[any]any:
			current = convertMapInterface(v)
		default:
			return nil, fmt.Errorf("key path %s does not point to an object", strings.Join(keys[:i+1], "."))
		}
	}

	return nil, fmt.Errorf("unexpected end of key path")
}

func (p *Parser) SetValue(data map[string]any, keyPath string, value any) error {
	keys := strings.Split(keyPath, ".")
	current := data

	for i, key := range keys {
		if i == len(keys)-1 {
			current[key] = value
			return nil
		}

		next, exists := current[key]
		if !exists {
			current[key] = make(map[string]any)
			next = current[key]
		}

		switch v := next.(type) {
		case map[string]any:
			current = v
		case map[any]any:
			converted := convertMapInterface(v)
			current[key] = converted
			current = converted
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
		
		keys = append(keys, fullKey)
		
		switch v := value.(type) {
		case map[string]any:
			subKeys := p.GetAllKeys(v, fullKey)
			keys = append(keys, subKeys...)
		case map[any]any:
			converted := convertMapInterface(v)
			subKeys := p.GetAllKeys(converted, fullKey)
			keys = append(keys, subKeys...)
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

func (p *Parser) ValidateKeyPath(data map[string]any, keyPath string) error {
	_, err := p.GetValue(data, keyPath)
	return err
}
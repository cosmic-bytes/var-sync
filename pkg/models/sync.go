package models

import "time"

type FileFormat string

const (
	FormatJSON FileFormat = "json"
	FormatYAML FileFormat = "yaml"
	FormatTOML FileFormat = "toml"
)

type SyncRule struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	SourceFile  string     `json:"source_file"`
	SourceKey   string     `json:"source_key"`
	TargetFile  string     `json:"target_file"`
	TargetKey   string     `json:"target_key"`
	Enabled     bool       `json:"enabled"`
	Created     time.Time  `json:"created"`
	LastSync    *time.Time `json:"last_sync,omitempty"`
}

type SyncEvent struct {
	RuleID    string    `json:"rule_id"`
	Timestamp time.Time `json:"timestamp"`
	OldValue  any       `json:"old_value"`
	NewValue  any       `json:"new_value"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

type Config struct {
	Rules   []SyncRule `json:"rules"`
	LogFile string     `json:"log_file"`
	Debug   bool       `json:"debug"`
}

func (f FileFormat) String() string {
	return string(f)
}

func DetectFormat(filepath string) FileFormat {
	switch {
	case len(filepath) >= 5 && filepath[len(filepath)-5:] == ".yaml":
		return FormatYAML
	case len(filepath) >= 4 && filepath[len(filepath)-4:] == ".yml":
		return FormatYAML
	case len(filepath) >= 5 && filepath[len(filepath)-5:] == ".toml":
		return FormatTOML
	case len(filepath) >= 5 && filepath[len(filepath)-5:] == ".json":
		return FormatJSON
	default:
		return FormatJSON
	}
}
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"var-sync/pkg/models"
)

type Manager struct {
	config   *models.Config
	filepath string
}

func New() *models.Config {
	return &models.Config{
		Rules:   make([]models.SyncRule, 0),
		LogFile: "var-sync.log",
		Debug:   false,
	}
}

func Load(configPath string) (*models.Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := New()
		if err := Save(cfg, configPath); err != nil {
			return nil, fmt.Errorf("failed to create config file: %w", err)
		}
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg models.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

func Save(cfg *models.Config, configPath string) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func NewManager(configPath string) (*Manager, error) {
	cfg, err := Load(configPath)
	if err != nil {
		return nil, err
	}

	return &Manager{
		config:   cfg,
		filepath: configPath,
	}, nil
}

func (m *Manager) Config() *models.Config {
	return m.config
}

func (m *Manager) Save() error {
	return Save(m.config, m.filepath)
}

func (m *Manager) AddRule(rule models.SyncRule) {
	m.config.Rules = append(m.config.Rules, rule)
}

func (m *Manager) RemoveRule(id string) {
	for i, rule := range m.config.Rules {
		if rule.ID == id {
			m.config.Rules = append(m.config.Rules[:i], m.config.Rules[i+1:]...)
			break
		}
	}
}

func (m *Manager) GetRule(id string) *models.SyncRule {
	for i, rule := range m.config.Rules {
		if rule.ID == id {
			return &m.config.Rules[i]
		}
	}
	return nil
}
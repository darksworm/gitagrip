package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gitagrip/internal/eventbus"
)

// Config represents the application configuration
type Config struct {
	Version   int                 `json:"version"`
	BaseDir   string              `json:"base_dir"`
	Groups    map[string][]string `json:"groups"`    // group name -> repo paths
	UISettings UISettings         `json:"ui"`
}

// UISettings represents UI-related configuration
type UISettings struct {
	ShowAheadBehind bool `json:"show_ahead_behind"`
	AutosaveOnExit  bool `json:"autosave_on_exit"`
}

// ConfigService handles configuration management
type ConfigService interface {
	Load() (*Config, error)
	Save(config *Config) error
	LoadFromPath(path string) (*Config, error)
	SaveToPath(config *Config, path string) error
}

// configService is the concrete implementation
type configService struct {
	bus      eventbus.EventBus
	filePath string
}

// NewConfigService creates a new config service
func NewConfigService() ConfigService {
	configDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to home directory
		configDir, err = os.UserHomeDir()
		if err != nil {
			configDir = "."
		}
		configDir = filepath.Join(configDir, ".config")
	}
	
	// Create gitagrip config directory
	gitagripDir := filepath.Join(configDir, "gitagrip")
	os.MkdirAll(gitagripDir, 0755)
	
	return &configService{
		filePath: filepath.Join(gitagripDir, "config.json"),
	}
}

// NewConfigServiceWithBus creates a config service with event bus support
func NewConfigServiceWithBus(bus eventbus.EventBus) ConfigService {
	cs := NewConfigService().(*configService)
	cs.bus = bus
	return cs
}

// Load loads the configuration from file
func (cs *configService) Load() (*Config, error) {
	// Check if config file exists
	if _, err := os.Stat(cs.filePath); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		cfg := DefaultConfig()
		
		// Publish ConfigLoaded event if bus is available
		if cs.bus != nil {
			cs.bus.Publish(eventbus.ConfigLoadedEvent{
				BaseDir: cfg.BaseDir,
				Groups:  cfg.Groups,
			})
		}
		
		return cfg, nil
	}
	
	// Read config file
	data, err := os.ReadFile(cs.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	// Parse config
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	
	// Initialize maps if nil
	if cfg.Groups == nil {
		cfg.Groups = make(map[string][]string)
	}
	
	// Publish ConfigLoaded event if bus is available
	if cs.bus != nil {
		cs.bus.Publish(eventbus.ConfigLoadedEvent{
			BaseDir: cfg.BaseDir,
			Groups:  cfg.Groups,
		})
	}
	
	return &cfg, nil
}

// Save saves the configuration to file
func (cs *configService) Save(config *Config) error {
	// Ensure config directory exists
	dir := filepath.Dir(cs.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(cs.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	// Publish ConfigSaved event if bus is available
	if cs.bus != nil {
		cs.bus.Publish(eventbus.ConfigSavedEvent{})
	}
	
	return nil
}

// LoadFromPath loads configuration from a specific path
func (cs *configService) LoadFromPath(path string) (*Config, error) {
	// Check if config file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}
	
	// Read config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	// Parse config
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	
	// Initialize maps if nil
	if cfg.Groups == nil {
		cfg.Groups = make(map[string][]string)
	}
	
	return &cfg, nil
}

// SaveToPath saves configuration to a specific path
func (cs *configService) SaveToPath(config *Config, path string) error {
	// Ensure config directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	// Try to get home directory for default base dir
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	
	return &Config{
		Version: 1,
		BaseDir: homeDir,
		Groups:  make(map[string][]string),
		UISettings: UISettings{
			ShowAheadBehind: true,
			AutosaveOnExit:  true,
		},
	}
}
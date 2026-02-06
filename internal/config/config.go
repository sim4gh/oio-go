package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// Config holds all configuration values
type Config struct {
	BaseURL      string `json:"baseurl,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	LoggedInAt   string `json:"logged_in_at,omitempty"`
	DefaultTTL   string `json:"default_ttl,omitempty"`
	Quiet        bool   `json:"quiet,omitempty"`
}

var (
	instance *Config
	once     sync.Once
	mu       sync.RWMutex
	filePath string
)

// AllowedKeys are keys that users can modify
var AllowedKeys = []string{"baseurl", "default_ttl", "quiet"}

// ProtectedKeys are read-only keys
var ProtectedKeys = []string{"id_token", "access_token", "refresh_token", "logged_in_at"}

// Load loads the configuration from disk
func Load() (*Config, error) {
	var err error
	once.Do(func() {
		filePath, err = GetConfigPath()
		if err != nil {
			return
		}

		instance = &Config{}

		data, readErr := os.ReadFile(filePath)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				// Config doesn't exist yet, that's fine
				return
			}
			err = readErr
			return
		}

		if len(data) > 0 {
			if jsonErr := json.Unmarshal(data, instance); jsonErr != nil {
				err = jsonErr
				return
			}
		}
	})

	return instance, err
}

// Get returns the current configuration
func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		cfg, _ := Load()
		return cfg
	}
	return instance
}

// Save persists the configuration to disk
func Save() error {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		return errors.New("config not loaded")
	}

	if filePath == "" {
		var err error
		filePath, err = GetConfigPath()
		if err != nil {
			return err
		}
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(instance, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0600)
}

// Set sets a configuration value
func Set(key, value string) error {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		instance = &Config{}
	}

	switch key {
	case "baseurl":
		instance.BaseURL = value
	case "id_token":
		instance.IDToken = value
	case "access_token":
		instance.AccessToken = value
	case "refresh_token":
		instance.RefreshToken = value
	case "logged_in_at":
		instance.LoggedInAt = value
	case "default_ttl":
		instance.DefaultTTL = value
	case "quiet":
		instance.Quiet = value == "true"
	default:
		return errors.New("unknown config key: " + key)
	}

	return Save()
}

// SetConfig updates the entire config at once and saves
func SetConfig(cfg *Config) error {
	mu.Lock()
	instance = cfg
	mu.Unlock()

	return Save()
}

// Clear removes all configuration
func Clear() error {
	mu.Lock()
	instance = &Config{}
	mu.Unlock()

	return Save()
}

// Path returns the config file path
func Path() string {
	if filePath == "" {
		filePath, _ = GetConfigPath()
	}
	return filePath
}

// IsProtectedKey checks if a key is protected
func IsProtectedKey(key string) bool {
	for _, k := range ProtectedKeys {
		if k == key {
			return true
		}
	}
	return false
}

// IsAllowedKey checks if a key is user-modifiable
func IsAllowedKey(key string) bool {
	for _, k := range AllowedKeys {
		if k == key {
			return true
		}
	}
	return false
}

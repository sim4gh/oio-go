package whatsapp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "modernc.org/sqlite"
)

// noopLogger implements waLog.Logger with no output
type noopLogger struct{}

func (noopLogger) Warnf(string, ...interface{})  {}
func (noopLogger) Errorf(string, ...interface{}) {}
func (noopLogger) Infof(string, ...interface{})  {}
func (noopLogger) Debugf(string, ...interface{}) {}
func (noopLogger) Sub(string) waLog.Logger       { return noopLogger{} }

// GetDBPath returns the platform-specific WhatsApp database path
func GetDBPath() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, "Library", "Application Support", "oio")

	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		configDir = filepath.Join(appData, "oio")

	default:
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			configHome = filepath.Join(home, ".config")
		}
		configDir = filepath.Join(configHome, "oio")
	}

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "whatsapp.db"), nil
}

// NewClient creates a new WhatsApp client backed by SQLite
func NewClient() (*whatsmeow.Client, error) {
	dbPath, err := GetDBPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get database path: %w", err)
	}

	var log noopLogger
	container, err := sqlstore.New(context.Background(), "sqlite", "file:"+dbPath+"?_pragma=foreign_keys(1)", log)
	if err != nil {
		return nil, fmt.Errorf("failed to create session store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	client := whatsmeow.NewClient(deviceStore, log)
	return client, nil
}

// IsLinked checks if a WhatsApp session database exists with data
func IsLinked() bool {
	dbPath, err := GetDBPath()
	if err != nil {
		return false
	}
	info, err := os.Stat(dbPath)
	if err != nil {
		return false
	}
	return info.Size() > 0
}

// DeleteDB removes the WhatsApp session database and related files
func DeleteDB() error {
	dbPath, err := GetDBPath()
	if err != nil {
		return err
	}
	// Remove WAL and SHM files (SQLite journal)
	os.Remove(dbPath + "-wal")
	os.Remove(dbPath + "-shm")
	return os.Remove(dbPath)
}

// FormatNumber cleans a phone number and returns a WhatsApp JID
func FormatNumber(number string) types.JID {
	clean := ""
	for _, c := range number {
		if c >= '0' && c <= '9' {
			clean += string(c)
		}
	}
	return types.JID{
		User:   clean,
		Server: types.DefaultUserServer,
	}
}

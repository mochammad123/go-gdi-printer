package engine

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// AppConfig stores persistent service settings
type AppConfig struct {
	Port           int    `json:"port"`
	DefaultPrinter string `json:"default_printer"`
}

var (
	configMu   sync.RWMutex
	currentCfg AppConfig
	configPath string
)

func getConfigFilePath() string {
	exePath, err := os.Executable()
	if err == nil {
		return filepath.Join(filepath.Dir(exePath), "config.json")
	}
	return "config.json"
}

// LoadConfig loads config from config.json or creates default if not exists
func LoadConfig(overridePort int) AppConfig {
	configMu.Lock()
	defer configMu.Unlock()

	configPath = getConfigFilePath()
	currentCfg = AppConfig{
		Port:           8080,
		DefaultPrinter: "",
	}

	data, err := os.ReadFile(configPath)
	if err == nil {
		var loaded AppConfig
		if err := json.Unmarshal(data, &loaded); err == nil {
			if loaded.Port > 0 {
				currentCfg.Port = loaded.Port
			}
			currentCfg.DefaultPrinter = loaded.DefaultPrinter
		}
	} else {
		// Save default config
		SaveConfigLocked(currentCfg)
	}

	if overridePort > 0 {
		currentCfg.Port = overridePort
	}

	return currentCfg
}

// GetConfig returns a copy of current config
func GetConfig() AppConfig {
	configMu.RLock()
	defer configMu.RUnlock()
	return currentCfg
}

// SaveConfig updates and writes the config to config.json
func SaveConfig(cfg AppConfig) error {
	configMu.Lock()
	defer configMu.Unlock()
	return SaveConfigLocked(cfg)
}

// SaveConfigLocked writes config without acquiring lock
func SaveConfigLocked(cfg AppConfig) error {
	currentCfg = cfg
	if configPath == "" {
		configPath = getConfigFilePath()
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		log.Printf("Gagal menulis config.json: %v\n", err)
		return err
	}
	return nil
}

package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/iamnotpirates/idlixdownloader/internal/models"
)

const (
	DefaultBaseURL = "https://z2.idlixku.com"
	DefaultSubLang = "Indonesian"
)

func GetAppDataDir() string {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		home, _ := os.UserHomeDir()
		localAppData = filepath.Join(home, "AppData", "Local")
	}
	dir := filepath.Join(localAppData, "iamnotpirates", "idlixdownloader")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func GetBinDir() string {
	dir := filepath.Join(GetAppDataDir(), "bin")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func GetConfigPath() string {
	// 1. Check local config.json in working directory
	if _, err := os.Stat("config.json"); err == nil {
		return "config.json"
	}
	return filepath.Join(GetAppDataDir(), "config.json")
}

func DefaultConfig() models.AppConfig {
	home, _ := os.UserHomeDir()
	moviesDir := filepath.Join(home, "Videos", "Movies")
	seriesDir := filepath.Join(home, "Videos", "TV Shows")

	return models.AppConfig{
		MoviesDir:          moviesDir,
		SeriesDir:          seriesDir,
		SubLang:            DefaultSubLang,
		ConfirmDownload:    false,
		BaseURL:            DefaultBaseURL,
		MaxConcurrentTasks: 1,
	}
}

func LoadConfig() models.AppConfig {
	cfg := DefaultConfig()
	cfgPath := GetConfigPath()

	data, err := os.ReadFile(cfgPath)
	if err == nil {
		_ = json.Unmarshal(data, &cfg)
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.SubLang == "" {
		cfg.SubLang = DefaultSubLang
	}
	if cfg.MaxConcurrentTasks <= 0 {
		cfg.MaxConcurrentTasks = 1
	}

	return cfg
}

func SaveConfig(cfg models.AppConfig) error {
	cfgPath := GetConfigPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, data, 0644)
}

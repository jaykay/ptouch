package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func initConfig() {
	dir := configDir()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(dir)
	viper.SetEnvPrefix("PTOUCH")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		// Config file is optional — no error if missing.
		debugf("config: %v", err)
	}
}

// configDir returns the platform-appropriate config directory for ptouch.
func configDir() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir != "" {
		return filepath.Join(dir, "ptouch")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ptouch")
}

// configFilePath returns the full path to the config file.
func configFilePath() string {
	return filepath.Join(configDir(), "config.yaml")
}

// saveConfig writes the current viper config to disk.
func saveConfig() error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	return viper.WriteConfigAs(configFilePath())
}

// resolveHost returns the effective printer host: flag > config.
func resolveHost() string {
	if flagHost != "" {
		return flagHost
	}
	return viper.GetString("host")
}

// resolveModel returns the effective printer model: flag > config > default.
func resolveModel() string {
	if rootCmd.PersistentFlags().Changed("model") {
		return flagModel
	}
	if m := viper.GetString("model"); m != "" {
		return m
	}
	return flagModel
}

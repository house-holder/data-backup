package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
	"time"
)

type DB struct {
	Source string `toml:"source"`
	Stage  string `toml:"stage"`
}

type DatabaseConfig struct {
	RemotePath       string        `toml:"remote_path"`
	RegistryPath     string        `toml:"registry_path"`
	ReportFreqDays   int64         `toml:"report_freq_days"`
	WarnNoBackupDays int64         `toml:"warn_no_backup_days"`
	Databases        map[string]DB `toml:"databases"`
}

func loadConfig(path string, dbCfg *DatabaseConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}
	err = toml.Unmarshal(data, dbCfg)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return nil
}

func getConfig() string {
	if path := os.Getenv("CFG_PATH"); path != "" {
		return path
	}
	return "/etc/data-backup/config.toml"
}

func shouldBackup(e *BackupEntry, r *Registry) bool {
	for _, backup := range r.Backups {
		if backup.Hash == e.Hash {
			return false
		}
	}
	return true
}

func stampFilename(t time.Time, dbName string) string {
	return fmt.Sprintf(
		"%d%.2d%.2d-%.2d%.2d-%s.db",
		t.Year(),
		int(t.Month()),
		t.Day(),
		t.Hour(),
		t.Minute(),
		dbName)
}

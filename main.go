package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/BurntSushi/toml"
)

type DBFile struct {
	Size int    `json:"size_bytes"`
	Hash string `json:"hash"`
}

type Registry struct {
	LastBackup int64             `json:"last_backup"`
	BytesUsed  int               `json:"bytes_used"`
	Backups    map[string]DBFile `json:"backups"`
}

type DBCfg struct {
	Source string `toml:"source"`
	Stage  string `toml:"stage"`
}

type Config struct {
	RegistryPath string           `toml:"registry_path"`
	RemotePath   string           `toml:"remote_path"`
	Databases    map[string]DBCfg `toml:"databases"`
}

func saveRegistry(r map[string]Registry, regPath string) error {
	data, err := json.MarshalIndent(r, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}
	err = os.WriteFile(regPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}
	return nil
}

func loadRegistry(path string, r *map[string]Registry) error {
	jsonData, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to read registry: %w", err)
	}
	if len(jsonData) == 0 {
		return nil
	}
	err = json.Unmarshal(jsonData, r)
	if err != nil {
		return fmt.Errorf("failed to unmarshal registry: %w", err)
	}
	return nil
}

func loadConfig(path string, c *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}
	err = toml.Unmarshal(data, c)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return nil
}

func getSHA256(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func getFileInfo(path string) (*DBFile, error) {
	var f DBFile
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f.Hash = getSHA256(bytes)
	f.Size = len(bytes)
	return &f, nil
}

func getConfig() string {
	if path := os.Getenv("CFG_PATH"); path != "" {
		fmt.Println("Using dev config")
		return path
	}
	return "/etc/data-backup/config.toml"
}

func loadInit() (*Config, *map[string]Registry) {
	var cfg Config
	var regMap map[string]Registry

	err := loadConfig(getConfig(), &cfg)
	if err != nil {
		log.Fatal(err)
	}
	err = loadRegistry(cfg.RegistryPath, &regMap)
	if err != nil {
		log.Fatal(err)
	}

	return &cfg, &regMap
}

func shouldBackup(f *DBFile, r *Registry) bool {
	for _, backup := range r.Backups { //nil backups = fallthrough
		if backup.Hash == f.Hash {
			return false
		}
	}
	return true
}

func fmtFilename(t time.Time, dbName string) string {
	timeStr := fmt.Sprintf("%d-%.2d-%.2d", t.Year(), int(t.Month()), t.Day())
	return fmt.Sprintf("%s-%s.db", timeStr, dbName)
}

func backupDatabase(dbName string, cfg *DBCfg, reg *Registry) error {
	t := time.Now()
	reg.LastBackup = t.Unix()
	filePath := fmt.Sprintf("%s/%s", cfg.Stage, fmtFilename(t, dbName))

	subcommand := fmt.Sprintf(".backup '%s'", filePath)
	cmd := exec.Command("sqlite3", cfg.Source, subcommand)
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}

func computeBackupSize(reg *Registry) {}

func main() {
	cfg, registryMap := loadInit()

	for dbName, dbCfg := range cfg.Databases {
		file, err := getFileInfo(dbCfg.Source)
		if err != nil {
			log.Fatalf("failed to get file info: %v", err)
		}
		registry, ok := (*registryMap)[dbName]
		if !ok || shouldBackup(file, &registry) {
			backupDatabase(dbName, &dbCfg, &registry)
		}
	}
}

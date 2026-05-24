package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"time"

	"github.com/BurntSushi/toml"
)

const maxDBEntries = 10

type BackupEntry struct {
	SizeBytes  int    `json:"size_bytes"`
	Epoch      int64  `json:"timestamp"`
	LocalPath  string `json:"local_path"`
	RemotePath string `json:"remote_path"`
	Hash       string `json:"hash"`
}

type Registry struct {
	LastBackup int64                  `json:"last_backup"`
	TotalBytes int                    `json:"total_bytes"`
	Backups    map[string]BackupEntry `json:"backups"`
}

func (r *Registry) update() {
	if len(r.Backups) > maxDBEntries {
		filename := ""
		var pruneEpoch int64 = math.MaxInt64

		for name, entry := range r.Backups {
			if entry.Epoch < pruneEpoch {
				pruneEpoch = entry.Epoch
				filename = name
			}
		}
		r.deleteBackup(filename, r.Backups[filename].RemotePath)
	}
	r.updateBytes()
}

func (r *Registry) updateBytes() {
	r.TotalBytes = 0
	for _, file := range r.Backups {
		r.TotalBytes += file.SizeBytes
	}
}

func (r *Registry) createBackup(name string, db *DB, remote string) error {
	tm := time.Now()
	entry, err := getFileInfo(db.Source)
	if err != nil {
		log.Fatalf("failed to get file info: %v", err)
	}
	if !shouldBackup(entry, r) {
		return nil
	}

	filename := datestampFilename(tm, name)

	entry.Epoch = tm.Unix()
	entry.LocalPath = fmt.Sprintf("%s%s", db.Stage, filename)
	entry.RemotePath = fmt.Sprintf("%s/%s", remote, filename)

	r.LastBackup = tm.Unix()
	r.Backups[filename] = *entry

	subCmdLocal := fmt.Sprintf(".backup '%s'", entry.LocalPath)
	cmdLocal := exec.Command("sqlite3", db.Source, subCmdLocal)
	out, err := cmdLocal.Output()
	if out != nil {
		fmt.Println(string(out))
	}
	if err != nil {
		return err
	}

	cmdRemote := exec.Command("rclone", "copy", entry.LocalPath, remote)
	out, err = cmdRemote.Output()
	if out != nil {
		fmt.Println(string(out))
	}
	if err != nil {
		return err
	}
	r.update()
	return nil
}

func (r *Registry) deleteBackup(filename string, remote string) error {
	err := os.Remove(r.Backups[filename].LocalPath)
	if err != nil {
		return fmt.Errorf("failed to remove local file: %v", err)
	}
	cmd := exec.Command("rclone", "deletefile", remote)
	out, err := cmd.Output()
	if out != nil {
		fmt.Println(string(out))
	}
	if err != nil {
		return err
	}
	delete(r.Backups, filename)
	return nil
}

type DB struct {
	Source string `toml:"source"`
	Stage  string `toml:"stage"`
}

type DatabaseConfig struct {
	RegistryPath string        `toml:"registry_path"`
	RemotePath   string        `toml:"remote_path"`
	Databases    map[string]DB `toml:"databases"`
}

func saveState(r map[string]*Registry, regPath string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}
	err = os.WriteFile(regPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}
	return nil
}

func loadState(path string, r *map[string]*Registry) error {
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

func getSHA256(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func getFileInfo(path string) (*BackupEntry, error) {
	var e BackupEntry

	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	e.Hash = getSHA256(bytes)
	e.SizeBytes = len(bytes)

	return &e, nil
}

func getConfig() string {
	if path := os.Getenv("CFG_PATH"); path != "" {
		fmt.Println("Using dev config")
		return path
	}
	return "/etc/data-backup/config.toml"
}

func loadInit() (*DatabaseConfig, *map[string]*Registry) {
	var dbCfg DatabaseConfig
	regMap := map[string]*Registry{}

	err := loadConfig(getConfig(), &dbCfg)
	if err != nil {
		log.Fatal(err)
	}
	err = loadState(dbCfg.RegistryPath, &regMap)
	if err != nil {
		log.Fatal(err)
	}

	return &dbCfg, &regMap
}

func shouldBackup(e *BackupEntry, r *Registry) bool {
	for _, backup := range r.Backups { //nil backups = fallthrough
		if backup.Hash == e.Hash {
			fmt.Println("found matching hash")
			return false
		}
	}
	return true
}

func datestampFilename(t time.Time, dbName string) string {
	timeStr := fmt.Sprintf("%d-%.2d-%.2d", t.Year(), int(t.Month()), t.Day())
	return fmt.Sprintf("%s-%s.db", timeStr, dbName)
}

func main() {
	cfg, registryMap := loadInit()

	for dbName, db := range cfg.Databases {
		remote := cfg.RemotePath + dbName

		registry, ok := (*registryMap)[dbName]
		if !ok {
			newRegistry := Registry{Backups: map[string]BackupEntry{}}
			(*registryMap)[dbName] = &newRegistry
			registry = &newRegistry
		}

		registry.createBackup(dbName, &db, remote)
	}

	err := saveState(*registryMap, cfg.RegistryPath)
	if err != nil {
		log.Fatalf("problem saving application state: %v", err)
	}
}

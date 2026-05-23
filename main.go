package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

type File struct {
	SizeBytes int    `json:"size_bytes"`
	Hash      string `json:"hash"`
}

type Registry struct {
	LastBackup int64           `json:"last_backup"`
	Backups    map[string]File `json:"backups"`
}

type DBCfg struct {
	Source string `json:"source"`
	Stage  string `json:"stage"`
}

type Config struct {
	Registry   string           `json:"registry_path"`
	RemotePath string           `json:"remote_path"`
	Databases  map[string]DBCfg `json:"databases"`
}

func loadRegistry(cfgPath string) map[string]Registry {
	var registry map[string]Registry
	jsonData, err := os.ReadFile(cfgPath)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(jsonData, &registry)
	if err != nil {
		panic(err)
	}
	return registry
}

func getSHA256(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func getFileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return getSHA256(data), nil
}

func getCfg() string {
	if path := os.Getenv("CFG_PATH"); path != "" {
		fmt.Println("Using dev config")
		return path
	}
	return "/etc/data-backup/config.toml"
}

func getRegistry() string {
	if path := os.Getenv("REGISTRY"); path != "" {
		fmt.Println("Using dev config")
		return path
	}
	return "/etc/data-backup/registry.json"
}

func main() {
	sites := loadSites(getRegistry())
	for k, v := range sites {
		fmt.Printf("Key: %s, val: %d\n", k, v)
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"time"
)

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

type State struct {
	LastReport int64                `json:"last_report"`
	Databases  map[string]*Registry `json:"databases"`
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

func (r *Registry) deleteBackup(filename string, remote string) error {
	err := os.Remove(r.Backups[filename].LocalPath)
	if err != nil {
		return fmt.Errorf("failed to remove local file: %v", err)
	}
	cmd := exec.Command("rclone", "deletefile", remote)
	out, err := cmd.Output()
	if len(out) > 0 {
		fmt.Printf("        out: %s\n", string(out))
	}
	if err != nil {
		return err
	}
	delete(r.Backups, filename)
	return nil
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

	filename := stampFilename(tm, name)

	entry.Epoch = tm.Unix()
	entry.LocalPath = fmt.Sprintf("%s%s", db.Stage, filename)
	entry.RemotePath = fmt.Sprintf("%s/%s", remote, filename)

	r.LastBackup = tm.Unix()
	r.Backups[filename] = *entry

	subCmdLocal := fmt.Sprintf(".backup '%s'", entry.LocalPath)
	cmdLocal := exec.Command("sqlite3", db.Source, subCmdLocal)
	out, err := cmdLocal.Output()
	if len(out) > 0 {
		fmt.Println(string(out))
	}
	if err != nil {
		return err
	}

	cmdRemote := exec.Command("rclone", "copy", entry.LocalPath, remote)
	out, err = cmdRemote.Output()
	if len(out) > 0 {
		fmt.Println(string(out))
	}
	if err != nil {
		return err
	}

	r.update()
	return nil
}

func (s *State) load(regPath string) error {
	jsonData, err := os.ReadFile(regPath)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to read registry: %w", err)
	}
	if len(jsonData) == 0 {
		return nil
	}
	err = json.Unmarshal(jsonData, s)
	if err != nil {
		return fmt.Errorf("failed to unmarshal registry: %w", err)
	}
	return nil
}

func (s *State) save(regPath string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}
	err = os.WriteFile(regPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}
	return nil
}

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const maxDBEntries = 15
const secsPerDay = 86400

type Reporter struct {
	warnings    []string
	lastReport  int64
	intervalSec int64
	noBackupSec int64
}

func (rpt *Reporter) warn(format string, args ...any) {
	rpt.warnings = append(rpt.warnings, fmt.Sprintf(format, args...))
}

func (rpt *Reporter) validate(dbName string, reg *Registry, remote string) {
	cmd := exec.Command("rclone", "ls", remote)
	out, err := cmd.Output()
	if err != nil {
		rpt.warn("[%s] failed to list remote %s: %v", dbName, remote, err)
	}
	remoteFiles := map[string]int{}
	for line := range strings.SplitSeq(
		strings.TrimSpace(string(out)), "\n",
	) {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		size, _ := strconv.Atoi(fields[0])
		remoteFiles[fields[1]] = size
	}
	for filename, entry := range reg.Backups {
		remoteSize, exists := remoteFiles[filename]
		if !exists {
			rpt.warn("[%s] registry entry missing: %s", dbName, filename)
			continue
		}
		if remoteSize != entry.SizeBytes {
			rpt.warn("[%s] size mismatch (%s): reg=%d remote=%d",
				dbName,
				filename,
				entry.SizeBytes,
				remoteSize,
			)
		}
	}
	for filename := range remoteFiles {
		if _, exists := reg.Backups[filename]; !exists {
			rpt.warn("[%s] remote file not in registry: %s", dbName, filename)
		}
	}
	if reg.LastBackup > 0 {
		timeSince := time.Since(time.Unix(reg.LastBackup, 0))
		threshold := time.Duration(rpt.noBackupSec) * time.Second
		if timeSince > threshold {
			days := int(timeSince.Hours() / 24)
			rpt.warn("[%s] last backup was %d days ago", dbName, days)
		}
	}
}

func (rpt *Reporter) broadcast(s *State) {
	tm := time.Now().Unix()
	if len(rpt.warnings) > 0 {
		for _, warning := range rpt.warnings {
			fmt.Println(warning)
		}
		rpt.warnings = []string{}
	}
	if (tm - rpt.lastReport) >= rpt.intervalSec {
		rpt.lastReport = tm
		s.LastReport = tm
		// report routine items: rclone size gdrive:backups/ parsing? other?
	}
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

func loadInit() (*DatabaseConfig, State) {
	var dbCfg DatabaseConfig
	var state State
	state.Databases = map[string]*Registry{}

	err := loadConfig(getConfig(), &dbCfg)
	if err != nil {
		log.Fatal(err)
	}
	err = state.load(dbCfg.RegistryPath)
	if err != nil {
		log.Fatal(err)
	}

	return &dbCfg, state
}

func main() {
	cfg, state := loadInit()
	reporter := Reporter{
		lastReport:  state.LastReport,
		intervalSec: cfg.ReportFreqDays * secsPerDay,
		noBackupSec: cfg.WarnNoBackupDays * secsPerDay,
	}

	for dbName, db := range cfg.Databases {
		remote := cfg.RemotePath + dbName
		registry, ok := state.Databases[dbName]
		if !ok {
			newRegistry := Registry{Backups: map[string]BackupEntry{}}
			state.Databases[dbName] = &newRegistry
			registry = &newRegistry
		}
		registry.createBackup(dbName, &db, remote)
	}
	for dbName := range cfg.Databases {
		remote := cfg.RemotePath + dbName
		registry, _ := state.Databases[dbName]
		reporter.validate(dbName, registry, remote)
	}

	reporter.broadcast(&state)

	err := state.save(cfg.RegistryPath)
	if err != nil {
		log.Fatalf("problem saving application state: %v", err)
	}
}

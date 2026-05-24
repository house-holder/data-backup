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

func (rpt *Reporter) routineReport(s *State, cfg *DatabaseConfig) {
	cmd := exec.Command("rclone", "size", cfg.RemotePath)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("routineReport: failed to get remote size: %v", err)
	} else {
		for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
			if strings.HasPrefix(line, "Total size:") {
				size := strings.TrimPrefix(line, "Total size: ")
				log.Printf("Total '%s' size: %s", cfg.RemotePath, size)
			}
		}
	}

	for dbName, reg := range s.Databases {
		if reg.LastBackup == 0 {
			log.Printf("Last '%s' backup: never", dbName)
		} else {
			log.Printf("Last '%s' backup: %s", dbName, relativeTime(reg.LastBackup))
		}
	}
}

func (rpt *Reporter) broadcast(s *State, cfg *DatabaseConfig, debug bool) {
	tm := time.Now().Unix()
	if len(rpt.warnings) > 0 {
		for _, warning := range rpt.warnings {
			fmt.Println(warning)
		}
		rpt.warnings = []string{}
	}
	if (tm-rpt.lastReport) >= rpt.intervalSec || debug {
		rpt.lastReport = tm
		s.LastReport = tm
		rpt.routineReport(s, cfg)
	}
}

func relativeTime(epoch int64) string {
	t := time.Unix(epoch, 0).Local()
	now := time.Now()

	daysSince := int(now.Sub(t).Hours() / 24)
	timeStr := t.Format("02 Jan 15:04 MST")

	switch daysSince {
	case 0:
		return fmt.Sprintf("today (%s)", timeStr)
	case 1:
		return fmt.Sprintf("yesterday (%s)", timeStr)
	default:
		return fmt.Sprintf("%d days ago (%s)", daysSince, timeStr)
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
	debug := cfg.Debug
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

	reporter.broadcast(&state, cfg, debug)

	err := state.save(cfg.RegistryPath)
	if err != nil {
		log.Fatalf("problem saving application state: %v", err)
	}
}

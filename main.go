package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

type File struct {
	SizeBytes int    `json:"size_bytes"`
	Hash      string `json:"hash"`
}

type Site struct {
	Name       string          `json:"name"`
	Path       string          `json:"path"`
	LastBackup int64           `json:"last_backup"`
	Backups    map[string]File `json:"backups"`
}

// TODO: plan functions
/*
getSha256 		-> takes filepath, returns hash string (async/threaded?)
loadMetadata 	-> loads JSON (default path) into memory
storeMetadata 	-> converts in-memory objects to written JSON
*/

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

func main() {
	_, _ = getFileHash("nonsense")

	fmt.Println(getSHA256([]byte("some test string")))
	// f1ebecbffecf3f8e4f60db92de00b600ee7b695c30f255463d55b36ba4ae35d6
	fmt.Println(getSHA256([]byte("any test string")))
	// 9271675f13b85ffee2af5c98a4145382579ef20a2a5cb1310756357b5267090a
}

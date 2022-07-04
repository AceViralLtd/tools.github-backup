package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"
)

const (
	ProgressFileName = "progress.json"
)

type ProgressEntry struct {
	Downloaded bool
	Archived   bool
	Uploaded   bool
}

var CurrentRunProgress map[string]*ProgressEntry

// UpdateProgress will store the run progress to file to allow for resuming failed runs
func UpdateProgress(config *Config, repoName string, progress ProgressEntry) {
	if entry, ok := CurrentRunProgress[repoName]; ok {
		entry.Downloaded = progress.Downloaded
		entry.Archived = progress.Archived
		entry.Uploaded = progress.Uploaded
	} else {
		CurrentRunProgress[repoName] = &ProgressEntry{
			Downloaded: progress.Downloaded,
			Archived:   progress.Archived,
			Uploaded:   progress.Uploaded,
		}
	}

	data, err := json.Marshal(CurrentRunProgress)
	if err != nil {
		log.Println("Failed to update progress")
		return
	}

	progressFile := path.Join(config.Path.DownloadPath(), ProgressFileName)
	_ = ioutil.WriteFile(progressFile, data, 0644)
}

// InitProgress will resume progress state from file (if appropriate)
func InitProgress(config *Config) {
	progressFile := path.Join(config.Path.DownloadPath(), ProgressFileName)
	CurrentRunProgress = make(map[string]*ProgressEntry)

	if _, err := os.Stat(progressFile); err != nil {
		return
	}

	data, err := ioutil.ReadFile(progressFile)
	if err == nil {
		_ = json.Unmarshal(data, &CurrentRunProgress)
	}
}

package util

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/aceviralltd/github-backup/internal/config"
)

var writer *os.File
var writerMux sync.Mutex

// WriteToLog will create and append to the log file for this run
func WriteToLog(cfg *config.Config, archiveId, description string, reportedError error) {
	writerMux.Lock()
	openWriter(cfg)

	writer.WriteString(fmt.Sprintf("%s,%s,%v\n", archiveId, description, reportedError))
	writerMux.Unlock()
}

// CloseLog will attempt to close the file handle for the log file
func CloseLog() {
	if writer == nil {
		return
	}

	writer.Close()
	writer = nil
}

func openWriter(cfg *config.Config) {
	if writer != nil {
		return
	}

	var err error
	os.MkdirAll(cfg.Path.LogDir, 0744)

	writer, err = os.OpenFile(cfg.Path.LogPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		CloseLog()
		log.Fatal("Failed to open log file")
	}

	writer.WriteString("Archive Id, Description, Error\n")
}

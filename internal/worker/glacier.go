package worker

import (
	"errors"
	"log"
	"os"

	"github.com/aceviralltd/github-backup/internal/config"
	"github.com/aceviralltd/github-backup/internal/service/aws"
	"github.com/aceviralltd/github-backup/internal/util"
)

// InitializeGlacearWorker setus up the environment for and starts off the glacier worker goroutine
func InitializeGlacearWorker(cfg *config.Config, bufferSize int) {
	logger := log.New(os.Stdout, "glac: ", log.LstdFlags)
	logger.Println("starting glacier worker")

	glacierQueue = make(chan QueueEntry, bufferSize)
	WaitGroup.Add(1)

	go glacierWorker(logger, cfg)
}

// glacierWorker is a goroutine that will take entries from a queue (channel) and ipload them to
// aws glacier concurrently
func glacierWorker(logger *log.Logger, cfg *config.Config) {
	for entry := range glacierQueue {
		progress, ok := config.CurrentRunProgress[*entry.Repo.Name]
		if !ok {
			progress = &config.ProgressEntry{
				Downloaded: true,
				Archived:   true,
				Uploaded:   false,
			}
		}

		if progress.Uploaded {
			continue
		}

		archivePath := cfg.Path.ArchivePath(entry.Repo)

		logger.Printf("opening %s", archivePath)

		file, err := os.Open(archivePath)
		if err != nil {
			logger.Println("failed to open archive")
			util.WriteToLog(cfg, "", entry.Description, errors.New("Failed to open archive"))
			continue
		}

		logger.Println("uploading")
		archiveId, err := aws.UploadToGlacier(
			cfg,
			file,
			entry.Description,
		)

		if err != nil {
			logger.Println("upload failed")
		} else {
			logger.Println("upload complete")

			// we don't actually need to keep any of the archives once they are uploaded
			os.Remove(archivePath)
		}

		util.WriteToLog(cfg, archiveId, entry.Description, err)

		progress.Uploaded = true
		config.UpdateProgress(cfg, *entry.Repo.Name, *progress)

	}

	logger.Println("shutting down")
	WaitGroup.Done()
}

// enqueueUpload handles the sending the job to the glacier worker
func enqueueGlacierUpload(logger *log.Logger, progress *config.ProgressEntry, entry QueueEntry) {
	if progress.Uploaded {
		return
	}

	logger.Println("adding to glacier queue")
	glacierQueue <- entry
}

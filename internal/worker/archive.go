package worker

import (
	"errors"
	"log"
	"os"

	"github.com/aceviralltd/github-backup/internal/config"
	"github.com/aceviralltd/github-backup/internal/util"
	"github.com/google/go-github/v34/github"
)

// InitializeArchiveWorker will setup then env for and start the archiver goroutine
func InitializeArchiveWorker(cfg *config.Config, bufferSize int) {
	logger := log.New(os.Stdout, "arc: ", log.LstdFlags)
	logger.Println("starting archive worker")

	ArciveQueue = make(chan QueueEntry, bufferSize)
	WaitGroup.Add(1)

	go archiveWorker(logger, cfg)
}

// archiveWorker handles the archiving of repos
func archiveWorker(logger *log.Logger, cfg *config.Config) {
	for entry := range ArciveQueue {
		if cfg.Github.SkipArchived && *entry.Repo.Archived {
			logger.Printf("skipping archived repo: %s", *entry.Repo.Name)
		}

		progress, ok := config.CurrentRunProgress[*entry.Repo.Name]
		if !ok {
			progress = &config.ProgressEntry{
				Downloaded: true,
				Archived:   false,
				Uploaded:   false,
			}
		}

		logger.Println("archiving", *entry.Repo.Name)
		if _, err := util.ArchiveDirectory(cfg, entry.Repo); err != nil {
			logger.Println("archive failed", err)
			util.WriteToLog(cfg, "", entry.Description, errors.New("Failed to archive repo"))
			return
		}

		progress.Archived = true
		config.UpdateProgress(cfg, *entry.Repo.Name, *progress)

		enqueueGlacierUpload(logger, progress, entry)
	}

	close(glacierQueue)
	WaitGroup.Done()
}

// enqueueArchive handles the zipping of the repo
func enqueueArchive(
	logger *log.Logger,
	repo *github.Repository,
	cfg *config.Config,
	progress *config.ProgressEntry,
	description string,
) {
	entry := QueueEntry{repo, description}

	if progress.Archived {
		enqueueGlacierUpload(logger, progress, entry)
		return
	}

	logger.Println("adding to archive queue")
	ArciveQueue <- entry
}

package worker

import (
	"errors"
	"log"

	"github.com/aceviralltd/github-backup/internal/config"
	githubService "github.com/aceviralltd/github-backup/internal/service/github"
	"github.com/aceviralltd/github-backup/internal/util"

	"github.com/google/go-github/v34/github"
)

// ProcessRepo will do all the work really, clone the repo, archive it then pass the path
// to the glacier worker
func ProcessRepo(logger *log.Logger, cfg *config.Config, repo *github.Repository) {
	progress, ok := config.CurrentRunProgress[*repo.Name]
	if !ok {
		progress = &config.ProgressEntry{}
	}

	// if the repo has already been uploaded to glacier then there is nothing to do
	if progress.Uploaded {
		return
	}

	logger.Printf("processing %s", *repo.Name)
	description := cfg.ArchiveDescription(*repo.Name)

	if !downloadRepo(logger, repo, cfg, progress, description) {
		return
	}

	enqueueArchive(logger, repo, cfg, progress, description)
}

// download handles the cloning of the repo
func downloadRepo(
	logger *log.Logger,
	repo *github.Repository,
	cfg *config.Config,
	progress *config.ProgressEntry,
	description string,
) bool {
	if progress.Downloaded {
		return true
	}

	logger.Println("cloning repo")
	if _, err := githubService.DownloadRepo(cfg, repo, logger); err != nil {
		logger.Println("clone failed: " + err.Error())
		util.WriteToLog(cfg, "", description, errors.New("Failed to clone repo: "+err.Error()))

		return false
	}

	progress.Downloaded = true
	config.UpdateProgress(cfg, *repo.Name, *progress)

	return true
}

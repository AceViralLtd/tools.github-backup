package main

import (
	"log"
	"os"

	"github.com/aceviralltd/github-backup/internal/config"
	"github.com/aceviralltd/github-backup/internal/service/aws"
	githubService "github.com/aceviralltd/github-backup/internal/service/github"
	"github.com/aceviralltd/github-backup/internal/util"
	"github.com/aceviralltd/github-backup/internal/worker"

	"github.com/indeedhat/gli"
)

const (
	ErrNone   = 0
	ErrConfig = 1
	ErrGithub = 2
	ErrAws    = 3
)

// GithubBackup is used by the gli framework to provide the cli application entry point
type GithubBackup struct {
	Date       string `gli:"date" description:"Overwrite the target date with the given one"`
	ConfigPath string `gli:"config" description:"Path to the config file"`
	Help       bool   `gli:"^help,h" description:"Show this document"`

	cfg *config.Config
}

// Run the command logic
func (cmd *GithubBackup) Run() int {
	var err error
	logger := log.New(os.Stdout, "main: ", log.LstdFlags)
	defer util.CloseLog()

	if !cmd.loadConfig(logger) {
		return ErrConfig
	}

	logger.Println("listing repos")
	repos, err := githubService.ListRepos(cmd.cfg)
	if err != nil {
		logger.Printf("ERROR: %s\n", err)
		return ErrGithub
	}

	if err = aws.CreateGlacierVault(cmd.cfg); err != nil {
		logger.Printf("ERROR: %s\n", err)
		return ErrAws
	}

	worker.InitializeArchiveWorker(cmd.cfg, len(repos))
	worker.InitializeGlacearWorker(cmd.cfg, len(repos))

	logger.Println("cloning repos")
	for _, repo := range repos {
		worker.ProcessRepo(logger, cmd.cfg, repo)
	}

	close(worker.ArciveQueue)
	worker.WaitGroup.Wait()

	return ErrNone
}

// NeedHelp makes the decision if the help document should be shown or not
func (cmd *GithubBackup) NeedHelp() bool {
	return cmd.Help
}

// loadConfig from file and setup the progress log
func (cmd *GithubBackup) loadConfig(logger *log.Logger) bool {
	var err error

	if cmd.cfg, err = config.LoadConfig(cmd.ConfigPath); err != nil {
		logger.Printf("ERROR: %s\n", err)
		return false
	}

	if cmd.Date != "" {
		cmd.cfg.ForceDate(cmd.Date)
	}

	config.InitProgress(cmd.cfg)

	return true
}

// main is well.. main, what do you want form me?
func main() {
	app := gli.NewApplication(&GithubBackup{}, "Archive github repos to aws glacier")
	app.Run()
}

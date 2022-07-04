package main

// This is not a full functional tool
// it is a hacky script i used as proof of concept to get an archive doewnloaded again
//
// In theory it should actually work if you fill out all of the const section properly
//
// Bare in mind this is a long running process, it is likely to take hours or days for the job
// to get processed before the download can begin

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/aceviralltd/github-backup/internal/config"
	awsService "github.com/aceviralltd/github-backup/internal/service/aws"
	"github.com/aceviralltd/github-backup/internal/util"
	"github.com/indeedhat/gli"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/glacier"
)

const (
	ErrNone   = 0
	ErrConfig = 1
	ErrGithub = 2
	ErrAws    = 3
)

const (
	AwsToken     = ""
	AwsSecret    = ""
	AwsUserId    = ""
	AwsAccountId = ""
	AwsRegion    = ""
	AwsVaultName = ""
	AwsArchiveId = ""
	AwsJobType   = ""

	OutputFilename = "output.zip"

	JobSleepTime = time.Minute * 5
)

var (
	awsGlacierClient *glacier.Client
	jobId            string
)

type BackupRestore struct {
	ConfigPath string `gli:"config" description:"Path to the config file"`
	Output     string `gli:"output,o" description:"The name of the archive to download to" default:"output.zip"`
	Help       bool   `gli:"^help,h" description:"Show this document"`
	ArchiveId  string `gli:"!" description:"The archive you want to download"`

	cfg   *config.Config
	jobId string
}

// Run the command logic
func (cmd *BackupRestore) Run() int {
	var err error
	ctx := context.Background()

	logger := log.New(os.Stdout, "main: ", log.LstdFlags)
	defer util.CloseLog()

	if !cmd.loadConfig(logger) {
		return ErrConfig
	}

	client, err := awsService.GlacierClient(ctx, cmd.cfg)
	if err != nil {
		panic(err)
	}

	cmd.jobId, err = awsService.InitArchiveDownload(cmd.cfg, cmd.ArchiveId)
	if err != nil {
		panic(err)
	}

	if cmd.jobId == "" {
		panic("Failed to create job")
	}

	cmd.awaitJobCompletion(ctx, client)
	cmd.downloadFile(ctx, client)

	return 0
}

// NeedHelp makes the decision if the help document should be shown or not
func (cmd *BackupRestore) NeedHelp() bool {
	return cmd.Help
}

// loadConfig from file and setup the progress log
func (cmd *BackupRestore) loadConfig(logger *log.Logger) bool {
	var err error

	if cmd.cfg, err = config.LoadConfig(cmd.ConfigPath); err != nil {
		logger.Printf("ERROR: %s\n", err)
		return false
	}

	config.InitProgress(cmd.cfg)

	return true
}

// awaitJobCompletion for glacier retrieval
func (cmd *BackupRestore) awaitJobCompletion(ctx context.Context, client *glacier.Client) {
	fmt.Println("Waiting for glacier job to complete")

	for range time.After(JobSleepTime) {
		if cmd.jobIsComplete(ctx, client, cmd.jobId) {
			fmt.Println("!!!! JOB COMPLETE !!!")
			return
		}

		fmt.Println("Still not complete, waiting for 5 minutes")
	}
}

// downloadFile from aws to the local machine
func (cmd *BackupRestore) downloadFile(ctx context.Context, client *glacier.Client) {
	output, err := client.GetJobOutput(ctx, &glacier.GetJobOutputInput{
		AccountId: aws.String(cmd.cfg.Aws.AccountId),
		JobId:     aws.String(cmd.jobId),
		VaultName: aws.String(cmd.cfg.Aws.Vault),
	})

	if err != nil {
		panic(err)
	}

	outFile, err := os.Create(OutputFilename)
	if err != nil {
		panic(err)
	}

	defer outFile.Close()

	if _, err = io.Copy(outFile, output.Body); err != nil {
		panic(err)
	}
}

// jobIsComplete checks if the given job id has finished processing on the aws servers
func (cmd *BackupRestore) jobIsComplete(ctx context.Context, client *glacier.Client, jobId string) bool {
	for _, job := range awsService.ListCurrentJobs(ctx, cmd.cfg, client).JobList {
		if *job.JobId != jobId {
			continue
		}

		return job.Completed
	}

	return false
}

func main() {
	app := gli.NewApplication(&BackupRestore{}, "Restore a backup from glacier")
	app.Run()
}

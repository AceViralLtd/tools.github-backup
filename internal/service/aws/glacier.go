package aws

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"

	"github.com/aceviralltd/github-backup/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/glacier"
	"github.com/aws/aws-sdk-go-v2/service/glacier/types"
	glacierV1 "github.com/aws/aws-sdk-go/service/glacier"
)

// 100MB
var MultipartChunkSizeHeader = "134217728"
var MultipartChunkSize int64 = 134217728

var awsGlacierClient *glacier.Client

// GlacierClient will setup the config and create a client connection for aws glacier
func GlacierClient(ctx context.Context, cfg *config.Config) (*glacier.Client, error) {
	if awsGlacierClient == nil {
		glacierConf, err := buildAwsConfig(cfg, ctx)
		if err != nil {
			return nil, err
		}

		awsGlacierClient = glacier.NewFromConfig(glacierConf, func(opts *glacier.Options) {
			opts.Region = cfg.Aws.Region
		})
	}

	return awsGlacierClient, nil
}

// CreateGlacierVault will setup a brand new vault for this run
func CreateGlacierVault(cfg *config.Config) error {
	ctx := context.Background()

	client, err := GlacierClient(ctx, cfg)
	if err != nil {
		return err
	}

	_, err = client.CreateVault(ctx, &glacier.CreateVaultInput{
		AccountId: &cfg.Aws.AccountId,
		VaultName: &cfg.Aws.Vault,
	})

	return err
}

// Upload the archive of the git dir to glacier
func UploadToGlacier(cfg *config.Config, file *os.File, description string) (string, error) {
	ctx := context.Background()

	client, err := GlacierClient(ctx, cfg)
	if err != nil {
		return "", err
	}

	stat, err := file.Stat()
	if err != nil {
		return "", err
	}

	if stat.Size() > MultipartChunkSize {
		return multiPartUpload(cfg, file, stat, client, description)
	}

	response, err := client.UploadArchive(
		ctx,
		&glacier.UploadArchiveInput{
			AccountId:          &cfg.Aws.AccountId,
			VaultName:          &cfg.Aws.Vault,
			ArchiveDescription: &description,
			Body:               file,
		},
	)

	if err != nil {
		return "", err
	}

	return *response.ArchiveId, err
}

// InitArchiveDownload will start the process of downloading an archive to the local machine
func InitArchiveDownload(cfg *config.Config, archiveId string) (string, error) {
	ctx := context.Background()

	client, err := GlacierClient(ctx, cfg)
	if err != nil {
		return "", err
	}

	job, err := client.InitiateJob(ctx, &glacier.InitiateJobInput{
		AccountId: aws.String(cfg.Aws.AccountId),
		VaultName: aws.String(cfg.Aws.Vault),
		JobParameters: &types.JobParameters{
			ArchiveId: &archiveId,
			Type:      aws.String("archive-retrieval"),
		},
	})

	if err != nil {
		return "", err
	}

	return *job.JobId, nil
}

// ListCurrentJobs on aws glacier
func ListCurrentJobs(ctx context.Context, cfg *config.Config, client *glacier.Client) *glacier.ListJobsOutput {
	output, err := client.ListJobs(ctx, &glacier.ListJobsInput{
		AccountId: aws.String(cfg.Aws.AccountId),
		VaultName: aws.String(cfg.Aws.Vault),
	})

	if err != nil {
		panic(err)
	}

	return output
}

// multiPartUpload will send the file in 100MB chunck to glacier
func multiPartUpload(
	cfg *config.Config,
	file *os.File,
	stat fs.FileInfo,
	client *glacier.Client,
	description string,
) (
	string,
	error,
) {
	ctx := context.Background()

	mpResponse, err := client.InitiateMultipartUpload(ctx, &glacier.InitiateMultipartUploadInput{
		AccountId:          &cfg.Aws.AccountId,
		VaultName:          &cfg.Aws.Vault,
		ArchiveDescription: &description,
		PartSize:           &MultipartChunkSizeHeader,
	})

	if err != nil {
		return "", err
	}

	var chunkErr error
	var start int64

	for start = 0; start < stat.Size(); start += MultipartChunkSize {
		end := int64(math.Min(float64(start+MultipartChunkSize), float64(stat.Size())))

		data := make([]byte, int(end-start))
		if _, err := file.Read(data); err != nil {
			chunkErr = err
			break
		}

		_, chunkErr = client.UploadMultipartPart(ctx, &glacier.UploadMultipartPartInput{
			AccountId: &cfg.Aws.AccountId,
			VaultName: &cfg.Aws.Vault,
			UploadId:  mpResponse.UploadId,
			Range:     aws.String(fmt.Sprintf("bytes %d-%d/*", start, end-1)),
			Body:      bytes.NewReader(data),
		})

		if chunkErr != nil {
			break
		}
	}

	if chunkErr != nil {
		_, _ = client.AbortMultipartUpload(ctx, &glacier.AbortMultipartUploadInput{
			AccountId: &cfg.Aws.AccountId,
			VaultName: &cfg.Aws.Vault,
			UploadId:  mpResponse.UploadId,
		})

		return "", chunkErr
	}

	_, _ = file.Seek(0, io.SeekStart)
	_, err = client.CompleteMultipartUpload(ctx, &glacier.CompleteMultipartUploadInput{
		AccountId:   &cfg.Aws.AccountId,
		VaultName:   &cfg.Aws.Vault,
		UploadId:    mpResponse.UploadId,
		ArchiveSize: aws.String(fmt.Sprint(stat.Size())),
		Checksum:    aws.String(hex.EncodeToString(glacierV1.ComputeHashes(file).TreeHash)),
	})

	if err != nil {
		return "", err
	}

	return *mpResponse.UploadId, nil
}

// awsSession will create a new aws session for the provided credentials
func buildAwsConfig(cfg *config.Config, ctx context.Context) (aws.Config, error) {
	return awsConfig.LoadDefaultConfig(
		ctx,
		awsConfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.Aws.UserId,
				cfg.Aws.Secret,
				cfg.Aws.Token,
			),
		),
	)
}

package util

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"

	"github.com/aceviralltd/github-backup/internal/config"

	"github.com/google/go-github/v34/github"
)

// ArchiveDirectory will archive and the remove the given directory
func ArchiveDirectory(cfg *config.Config, repo *github.Repository) (string, error) {
	archivePath := cfg.Path.ArchivePath(repo)
	directoryPath := cfg.Path.RepoPath(repo)

	file, err := os.Create(archivePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	zipper := zip.NewWriter(file)
	defer zipper.Close()

	err = filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		zipFile, err := zipper.Create(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(zipFile, file)
		return err
	})

	if err != nil {
		os.RemoveAll(archivePath)
		return "", err
	}

	os.RemoveAll(directoryPath)
	return archivePath, nil
}

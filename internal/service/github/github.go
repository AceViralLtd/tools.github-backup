package github

import (
	"context"
	"fmt"
	"log"
	"os/exec"

	"github.com/aceviralltd/github-backup/internal/config"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v34/github"
)

var githubApiClient *github.Client

// ListRepos will return a full list of the repos in the organisation
func ListRepos(cfg *config.Config) ([]*github.Repository, error) {
	var repoList []*github.Repository

	client := githubClient(cfg)
	ctx := context.Background()

	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, cfg.Github.OrgName, opt)
		if err != nil {
			return nil, err
		}

		repoList = append(repoList, repos...)
		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return repoList, nil
}

// DownloadRepo will attempt to clone a repo
func DownloadRepo(cfg *config.Config, repo *github.Repository, logger *log.Logger) (string, error) {
	// if there is no download url it means that the repo is empty so we can just skip it
	if repo.GetArchiveURL() == "" {
		return "", nil
	}

	_, err := git.PlainClone(cfg.Path.RepoPath(repo), true, &git.CloneOptions{
		Auth: &http.BasicAuth{
			Username: cfg.Github.Username,
			Password: cfg.Github.Password,
		},
		URL: repo.GetCloneURL(),
	})

	if err != nil {
		if cfg.GitBin != "" {
			logger.Printf("Failed with error: %s - Falling back to shell clone\n", err)
			return downloadRepoFallback(cfg, repo)
		}

		return "", err
	}

	return cfg.Path.RepoPath(repo), nil
}

// downloadRepoFallback will only be called if the standard clone/download fails and the conf.GitBin var is set
//
// It will attempt to use the git cli application to do the clone instead of the go lib
func downloadRepoFallback(cfg *config.Config, repo *github.Repository) (string, error) {
	cloneUrl := fmt.Sprintf(
		"https://%s:%s@github.com/%s/%s",
		cfg.Github.Username,
		cfg.Github.Password,
		cfg.Github.OrgName,
		repo.GetName(),
	)

	cmd := exec.Command(
		cfg.GitBin,
		"clone",
		cloneUrl,
		cfg.Path.RepoPath(repo),
		"--bare",
	)

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return cfg.Path.RepoPath(repo), nil
}

func githubClient(cfg *config.Config) *github.Client {
	if githubApiClient == nil {
		auth := github.BasicAuthTransport{
			Username: cfg.Github.Username,
			Password: cfg.Github.Password,
		}

		githubApiClient = github.NewClient(auth.Client())
	}

	return githubApiClient
}

package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/go-github/v34/github"
	"github.com/pelletier/go-toml"
)

const (
	ConfigFile        = ".ghb.toml"
	DefaultRoodDir    = "backup"
	DefaultLogDir     = "logs"
	DefaultDateFormat = "2006-01-02"
)

type Config struct {
	Github githubConfig
	Path   pathConfig
	Aws    awsConfig

	GitBin string `toml:"git_bin"`
}

// ArchiveDescription will generate a unique description for the given repo
func (c *Config) ArchiveDescription(repo string) string {
	return fmt.Sprintf(
		"%s - %s",
		c.Path.date(),
		repo,
	)
}

// ForceDate sets the PathConfig.ForceDate string to force the tool to use a specific date
func (c *Config) ForceDate(dateString string) {
	c.Path.ForceDate = dateString
}

type githubConfig struct {
	Username     string
	Password     string
	OrgName      string `toml:"org_name"`
	SkipArchived bool   `toml:"skip_archived"`
}

type awsConfig struct {
	Token     string
	Secret    string
	UserId    string `toml:"user_id"`
	Region    string
	AccountId string `toml:"account_id"`
	Vault     string
}

type pathConfig struct {
	RootDir    string `toml:"root_dir"`
	DateFormat string `toml:"date_format" default:"2006-01-02"`
	LogDir     string `toml:"log_dir" default:"logs"`
	ForceDate  string
}

// date will return the appropriate date string for the run
func (c pathConfig) date() string {
	if c.ForceDate != "" {
		return c.ForceDate
	}

	return time.Now().Format(c.DateFormat)
}

// DownloadPath will build up a root path for storing downloads in
func (c pathConfig) DownloadPath() string {
	return path.Join(c.RootDir, c.date())
}

// RepoPath will build up a download loaction for the repo based on iteslf and the config
func (c pathConfig) RepoPath(repo *github.Repository) string {
	return path.Join(c.DownloadPath(), *repo.Name)
}

// ArchivePath will build up a full path to the final archive for the repo
func (c pathConfig) ArchivePath(repo *github.Repository) string {
	return path.Join(
		c.DownloadPath(),
		fmt.Sprintf("%s.zip", *repo.Name),
	)
}

// LogPath will build up a path for this run of the archiver
func (c pathConfig) LogPath() string {
	return path.Join(
		c.LogDir,
		fmt.Sprintf("%s.log", c.date()),
	)
}

// LoadConfig unsuprisingly tries to load the config form file
func LoadConfig(pathOverride string) (*Config, error) {
	data, err := ioutil.ReadFile(negotiateConfigPath(pathOverride))
	if err != nil {
		return nil, err
	}

	conf := &Config{}

	if err = toml.Unmarshal(data, conf); err != nil {
		return nil, err
	}

	if err = fillConfigDefaults(conf); err != nil {
		return nil, err
	}

	return conf, err
}

// negotiateConfigPath will build the most appropriate path to the config file it can
func negotiateConfigPath(pathOverride string) string {
	if pathOverride != "" {
		return pathOverride
	}

	pwd, err := os.Getwd()
	if err != nil {
		return ConfigFile
	}

	return path.Join(pwd, ConfigFile)
}

// fillConfigDefaults will fill out default values on the config as well as expanding paths
func fillConfigDefaults(config *Config) error {
	if config.Path.RootDir == "" {
		pwd, err := os.Getwd()
		if err != nil {
			return err
		}

		config.Path.RootDir = path.Join(pwd, DefaultRoodDir)
	} else if strings.HasPrefix(config.Path.RootDir, "./") {
		pwd, err := os.Getwd()
		if err != nil {
			return err
		}

		config.Path.RootDir = path.Join(pwd, config.Path.RootDir[2:])
	} else if strings.HasPrefix(config.Path.RootDir, "~") {
		home := os.Getenv("HOME")

		config.Path.RootDir = path.Join(home, config.Path.RootDir[1:])
	}

	if config.Path.LogDir == "" {
		pwd, err := os.Getwd()
		if err != nil {
			return err
		}

		config.Path.LogDir = path.Join(pwd, DefaultLogDir)
	} else if strings.HasPrefix(config.Path.LogDir, "./") {
		pwd, err := os.Getwd()
		if err != nil {
			return err
		}

		config.Path.LogDir = path.Join(pwd, config.Path.LogDir[2:])
	} else if strings.HasPrefix(config.Path.LogDir, "~") {
		home := os.Getenv("HOME")

		config.Path.LogDir = path.Join(home, config.Path.LogDir[1:])
	}

	config.Aws.Vault = fmt.Sprintf("%s_%s", config.Aws.Vault, config.Path.date())

	return nil
}

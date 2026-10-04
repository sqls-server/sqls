package config

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/sqls-server/sqls/internal/database"
	"gopkg.in/yaml.v2"
)

var (
	ErrNotFoundConfig = errors.New("NotFound Config")
)

var (
	YamlConfigPath = configFilePath("config.yml")
)

var (
	DefaultConfigFiles = []string{
		"config.yml",
		"config.yaml",
		"config.toml",
	}

	WorkspaceConfigFiles = []string{
		".sqls.yml",
		".sqls.yaml",
		".sqls.toml",
		"sqls.yml",
		"sqls.yaml",
		"sqls.toml",
		filepath.Join(".config", "sqls", "config.yml"),
		filepath.Join(".config", "sqls", "config.yaml"),
		filepath.Join(".config", "sqls", "config.toml"),
		filepath.Join(".config", "sqls.yml"),
		filepath.Join(".config", "sqls.yaml"),
		filepath.Join(".config", "sqls.toml"),
	}
)

type Config struct {
	LowercaseKeywords bool                 `json:"lowercaseKeywords" yaml:"lowercaseKeywords" toml:"lowercaseKeywords"`
	Connections       []*database.DBConfig `json:"connections" yaml:"connections" toml:"connections"`
}

func (c *Config) Validate() error {
	if len(c.Connections) > 0 {
		return c.Connections[0].Validate()
	}
	return nil
}

func NewConfig() *Config {
	cfg := &Config{}
	cfg.LowercaseKeywords = false
	return cfg
}

func FindDefaultConfigPath() string {
	for _, fn := range DefaultConfigFiles {
		fp := configFilePath(fn)
		if IsFileExist(fp) {
			return fp
		}
	}
	return ""
}

func FindWorkspaceConfigPath(workspaceDir string) string {
	if workspaceDir == "" {
		workspaceDir = "."
	}
	for _, fn := range WorkspaceConfigFiles {
		fp := filepath.Join(workspaceDir, fn)
		if IsFileExist(fp) {
			return fp
		}
	}
	return ""
}

func GetWorkspaceConfig(workspaceDir string) (*Config, error) {
	fp := FindWorkspaceConfigPath(workspaceDir)
	if fp == "" {
		return nil, ErrNotFoundConfig
	}
	return GetConfig(fp)
}

func GetDefaultConfig() (*Config, error) {
	fp := FindDefaultConfigPath()
	if fp == "" {
		fp = YamlConfigPath
	}
	cfg := NewConfig()
	if err := cfg.Load(fp); err != nil {
		return nil, err
	}
	return cfg, nil
}

func GetConfig(fp string) (*Config, error) {
	cfg := NewConfig()
	expandPath, err := expand(fp)
	if err != nil {
		return nil, err
	}
	if err := cfg.Load(expandPath); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Load(fp string) error {
	if !IsFileExist(fp) {
		return ErrNotFoundConfig
	}

	file, err := os.OpenFile(fp, os.O_RDONLY, 0666)
	if err != nil {
		return fmt.Errorf("cannot open config, %w", err)
	}
	defer file.Close()

	b, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("cannot read config, %w", err)
	}

	ext := strings.ToLower(filepath.Ext(fp))
	switch ext {
	case ".toml":
		if err = toml.Unmarshal(b, c); err != nil {
			return fmt.Errorf("failed unmarshal toml, %w, %s", err, string(b))
		}
	case ".yml", ".yaml":
		if err = yaml.Unmarshal(b, c); err != nil {
			return fmt.Errorf("failed unmarshal yaml, %w, %s", err, string(b))
		}
	default:
		if err = yaml.Unmarshal(b, c); err != nil {
			if tomlErr := toml.Unmarshal(b, c); tomlErr == nil {
				err = nil
			} else {
				return fmt.Errorf("failed unmarshal config, %w, %s", err, string(b))
			}
		}
	}

	if err := c.Validate(); err != nil {
		return fmt.Errorf("failed validation, %w", err)
	}
	return nil
}

func IsFileExist(fPath string) bool {
	_, err := os.Stat(fPath)
	return err == nil || !os.IsNotExist(err)
}

func configFilePath(fileName string) string {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "sqls", fileName)
	}

	var configDir string
	if runtime.GOOS == "darwin" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		configDir = filepath.Join(homeDir, ".config")
	} else {
		var err error
		configDir, err = os.UserConfigDir()
		if err != nil {
			panic(err)
		}
	}

	return filepath.Join(configDir, "sqls", fileName)
}

func expand(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, path[1:]), nil
}

func URItoPath(uriStr string) string {
	if uriStr == "" {
		return ""
	}
	u, err := url.Parse(uriStr)
	if err != nil || u.Scheme != "file" {
		return uriStr
	}
	path := u.Path
	if runtime.GOOS == "windows" && strings.HasPrefix(path, "/") {
		path = strings.TrimPrefix(path, "/")
	}
	return filepath.FromSlash(path)
}

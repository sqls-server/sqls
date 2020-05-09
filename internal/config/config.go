package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/lighttiger2505/sqls/internal/database"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v2"
)

var ymlConfigPath = configFilePath("config.yml")

type Config struct {
	Connections []*database.Config `json:"connections" yaml:"connections"`
}

func newConfig() *Config {
	cfg := &Config{}
	return cfg
}

func GetConfig() (*Config, error) {
	cfg := newConfig()
	if err := cfg.Load(ymlConfigPath); err != nil {
		return nil, err
	}
	return cfg, nil
}

func GetConfigWithPath(fp string) (*Config, error) {
	cfg := newConfig()
	if err := cfg.Load(fp); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Load(fp string) error {
	file, err := os.OpenFile(fp, os.O_RDONLY, 0666)
	if err != nil {
		return xerrors.Errorf("cannot open config, %+v", err)
	}
	defer file.Close()

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return xerrors.Errorf("cannot read config, %+v", err)
	}

	if err = yaml.Unmarshal(b, c); err != nil {
		return xerrors.Errorf("failed unmarshal yaml, %+v", err, string(b))
	}
	return nil
}

func IsFileExist(fPath string) bool {
	_, err := os.Stat(fPath)
	return err == nil || !os.IsNotExist(err)
}

func configFilePath(fileName string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(homeDir, ".config", "sqls", fileName)
}

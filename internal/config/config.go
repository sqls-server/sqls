package config

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/lighttiger2505/sqls/internal/database"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v2"
)

var ymlConfigPath = configFilePath("config.yml")
var jsonConfigPath = configFilePath("config.json")

type Config struct {
	Connections []*database.Config `json:"connections" yaml:"connections"`
}

func newConfig() *Config {
	cfg := &Config{}
	return cfg
}

func GetConfig() (*Config, error) {
	cfg := newConfig()
	if err := cfg.Load(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Path() string {
	return ymlConfigPath
}

func (c *Config) Read() (string, error) {
	if err := os.MkdirAll(filepath.Dir(ymlConfigPath), 0700); err != nil {
		return "", xerrors.Errorf("cannot create directory, %+v", err)
	}

	if !IsFileExist(ymlConfigPath) {
		_, err := os.Create(ymlConfigPath)
		if err != nil {
			return "", xerrors.Errorf("cannot create config, %+v", err.Error())
		}
	}

	file, err := os.OpenFile(ymlConfigPath, os.O_RDONLY, 0666)
	if err != nil {
		return "", xerrors.Errorf("cannot open config, %+v", err)
	}
	defer file.Close()

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return "", xerrors.Errorf("cannot read config, %+v", err)
	}

	return string(b), nil
}

func (c *Config) Load() error {
	if err := os.MkdirAll(filepath.Dir(ymlConfigPath), 0700); err != nil {
		return xerrors.Errorf("cannot create directory, %+v", err)
	}

	if !IsFileExist(ymlConfigPath) {
		if err := createNewConfig(); err != nil {
			return err
		}
	}

	file, err := os.OpenFile(ymlConfigPath, os.O_RDONLY, 0666)
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

func (c *Config) Save() error {
	file, err := os.OpenFile(ymlConfigPath, os.O_WRONLY, 0666)
	if err != nil {
		return xerrors.Errorf("cannot open file, %+v", err)
	}
	defer file.Close()

	out, err := yaml.Marshal(c)
	if err != nil {
		return xerrors.Errorf("cannot marshal config, %+v", err)
	}

	if _, err = io.WriteString(file, string(out)); err != nil {
		return xerrors.Errorf("cannot write config file, %+v", err)
	}
	return nil
}

func createNewConfig() error {
	// Create new config file
	_, err := os.Create(ymlConfigPath)
	if err != nil {
		return xerrors.Errorf("cannot create config, %+v", err)
	}

	// Add default settings
	cfg := newConfig()
	cfg.Connections = []*database.Config{
		{
			Driver:         "mysql",
			DataSourceName: "",
			Proto:          "tcp",
			User:           "root",
			Passwd:         "root",
			Host:           "127.0.0.1",
			Port:           13306,
			Path:           "",
			DBName:         "world",
			Params: map[string]string{
				"tls":        "skip-verify",
				"autocommit": "true",
			},
		},
	}
	if err := cfg.Save(); err != nil {
		return err
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

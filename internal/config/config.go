package config

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/lighttiger2505/sqls/internal/database"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v2"
)

var yamlConfigPath = filepath.Join(getXDGConfigPath(runtime.GOOS), "config.yml")
var jsonConfigPath = filepath.Join(getXDGConfigPath(runtime.GOOS), "config.json")

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
	return yamlConfigPath
}

func (c *Config) Read() (string, error) {
	if err := os.MkdirAll(filepath.Dir(yamlConfigPath), 0700); err != nil {
		return "", xerrors.Errorf("cannot create directory, %+v", err)
	}

	if !IsFileExist(yamlConfigPath) {
		_, err := os.Create(yamlConfigPath)
		if err != nil {
			return "", xerrors.Errorf("cannot create config, %+v", err.Error())
		}
	}

	file, err := os.OpenFile(yamlConfigPath, os.O_RDONLY, 0666)
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
	if err := os.MkdirAll(filepath.Dir(yamlConfigPath), 0700); err != nil {
		return xerrors.Errorf("cannot create directory, %+v", err)
	}

	if !IsFileExist(yamlConfigPath) {
		if err := createNewConfig(); err != nil {
			return err
		}
	}

	file, err := os.OpenFile(yamlConfigPath, os.O_RDONLY, 0666)
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
	file, err := os.OpenFile(yamlConfigPath, os.O_WRONLY, 0666)
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
	_, err := os.Create(yamlConfigPath)
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
	cfg.Save()
	return nil
}

func IsFileExist(fPath string) bool {
	_, err := os.Stat(fPath)
	return err == nil || !os.IsNotExist(err)
}

const AppName = "sqls"

func getXDGConfigPath(goos string) string {
	var dir string
	if goos == "windows" {
		dir = os.Getenv("APPDATA")
		if dir == "" {
			dir = filepath.Join(os.Getenv("USERPROFILE"), "Application Data", AppName)
		}
		dir = filepath.Join(dir, "lab")
	} else {
		dir = filepath.Join(os.Getenv("HOME"), ".config", AppName)
	}
	return dir
}

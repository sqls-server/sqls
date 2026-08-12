package database

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/sqls-server/sqls/dialect"
	"golang.org/x/crypto/ssh"
)

// azureAuthDrivers are the drivers that can log in with an Azure AD token.
var azureAuthDrivers = []dialect.DatabaseDriver{
	dialect.DatabaseDriverMySQL,
	dialect.DatabaseDriverMySQL8,
	dialect.DatabaseDriverMySQL57,
	dialect.DatabaseDriverMySQL56,
	dialect.DatabaseDriverPostgreSQL,
	dialect.DatabaseDriverMssql,
}

type Proto string

const (
	ProtoTCP  Proto = "tcp"
	ProtoUDP  Proto = "udp"
	ProtoUnix Proto = "unix"
	ProtoHTTP Proto = "http"
)

type DBConfig struct {
	Alias          string                 `json:"alias" yaml:"alias"`
	Driver         dialect.DatabaseDriver `json:"driver" yaml:"driver"`
	DataSourceName string                 `json:"dataSourceName" yaml:"dataSourceName"`
	Proto          Proto                  `json:"proto" yaml:"proto"`
	User           string                 `json:"user" yaml:"user"`
	Passwd         string                 `json:"passwd" yaml:"passwd"`
	Host           string                 `json:"host" yaml:"host"`
	Port           int                    `json:"port" yaml:"port"`
	Path           string                 `json:"path" yaml:"path"`
	DBName         string                 `json:"dbName" yaml:"dbName"`
	Params         map[string]string      `json:"params" yaml:"params"`
	SSHCfg         *SSHConfig             `json:"sshConfig" yaml:"sshConfig"`
	AzureAuth      *AzureAuthConfig       `json:"azureAuth" yaml:"azureAuth"`
}

func (c *DBConfig) Validate() error {
	if c.Driver == "" {
		return errors.New("required: connections[].driver")
	}

	if c.AzureAuth != nil {
		if !slices.Contains(azureAuthDrivers, c.Driver) {
			return fmt.Errorf("invalid: connections[].azureAuth is not supported by driver %s", c.Driver)
		}
		if err := c.AzureAuth.Validate(); err != nil {
			return err
		}
	}

	switch c.Driver {
	case
		dialect.DatabaseDriverMySQL,
		dialect.DatabaseDriverMySQL8,
		dialect.DatabaseDriverMySQL57,
		dialect.DatabaseDriverMySQL56,
		dialect.DatabaseDriverPostgreSQL,
		dialect.DatabaseDriverVertica:
		if c.DataSourceName == "" && c.Proto == "" {
			return errors.New("required: connections[].dataSourceName or connections[].proto")
		}

		if c.DataSourceName == "" && c.Proto != "" {
			if c.User == "" {
				return errors.New("required: connections[].user")
			}
			switch c.Proto {
			case ProtoTCP, ProtoUDP, ProtoHTTP:
				if c.Host == "" {
					return errors.New("required: connections[].host")
				}
			case ProtoUnix:
				if c.Path == "" {
					return errors.New("required: connections[].path")
				}
			default:
				return errors.New("invalid: connections[].proto")
			}
			if c.SSHCfg != nil {
				return c.SSHCfg.Validate()
			}
		}
	case dialect.DatabaseDriverSQLite3:
	case dialect.DatabaseDriverH2:
		if c.DataSourceName == "" {
			return errors.New("required: connections[].dataSourceName")
		}
	case dialect.DatabaseDriverMssql:
		if c.DataSourceName == "" && c.Proto == "" {
			return errors.New("required: connections[].dataSourceName or connections[].proto")
		}
		if c.DataSourceName == "" && c.Proto != "" {
			// With Azure AD auth the login name comes from the access token.
			if c.User == "" && c.AzureAuth == nil {
				return errors.New("required: connections[].user")
			}
			switch c.Proto {
			case ProtoTCP:
				if c.Host == "" {
					return errors.New("required: connections[].host")
				}
			case ProtoUDP, ProtoUnix, ProtoHTTP:
			default:
				return errors.New("invalid: connections[].proto")
			}
		}
	case dialect.DatabaseDriverOracle:
		if c.DataSourceName == "" && c.Proto == "" {
			return errors.New("required: connections[].dataSourceName or connections[].proto")
		}
		if c.DataSourceName == "" {
			if c.User == "" {
				return errors.New("required: connections[].user")
			}
			if c.Passwd == "" {
				return errors.New("required: connections[].Passwd")
			}
			if c.Host == "" {
				return errors.New("required: connections[].Host")
			}
			if c.Port <= 0 {
				return errors.New("required: connections[].Port")
			}
			if c.DBName == "" {
				return errors.New("required: connections[].DBName")
			}
		}
	case dialect.DatabaseDriverClickhouse:
		if c.DataSourceName == "" && c.Proto == "" {
			return errors.New("required: connections[].dataSourceName or connections[].proto")
		}

		if c.DataSourceName == "" && c.Proto != "" {
			if c.User == "" {
				return errors.New("required: connections[].user")
			}
			switch c.Proto {
			case ProtoTCP, ProtoHTTP:
				if c.Host == "" {
					return errors.New("required: connections[].host")
				}
			case ProtoUDP, ProtoUnix:
			default:
				return errors.New("invalid: connections[].proto")
			}
			if c.SSHCfg != nil {
				return c.SSHCfg.Validate()
			}
		}

	default:
		return errors.New("invalid: connections[].driver")
	}
	return nil
}

type SSHConfig struct {
	Host       string `json:"host" yaml:"host"`
	Port       int    `json:"port" yaml:"port"`
	User       string `json:"user" yaml:"user"`
	PassPhrase string `json:"passPhrase" yaml:"passPhrase"`
	PrivateKey string `json:"privateKey" yaml:"privateKey"`
}

func (s *SSHConfig) Validate() error {
	if s.Host == "" {
		return errors.New("required: connections[]sshConfig.host")
	}
	if s.User == "" {
		return errors.New("required: connections[].sshConfig.user")
	}
	if s.PrivateKey == "" {
		return errors.New("required: connections[].sshConfig.privateKey")
	}
	return nil
}

func (s *SSHConfig) Endpoint() string {
	port := s.Port
	if port == 0 {
		port = 22
	}
	return fmt.Sprintf("%s:%d", s.Host, port)
}

// Dial opens an ssh client connection that database drivers can tunnel through.
func (s *SSHConfig) Dial() (*ssh.Client, error) {
	sshConfig, err := s.ClientConfig()
	if err != nil {
		return nil, err
	}
	conn, err := ssh.Dial("tcp", s.Endpoint(), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot ssh dial, %w", err)
	}
	return conn, nil
}

func (s *SSHConfig) ClientConfig() (*ssh.ClientConfig, error) {
	buffer, err := os.ReadFile(s.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("cannot read SSH private key file, PrivateKey=%s, %w", s.PrivateKey, err)
	}

	var key ssh.Signer
	if s.PassPhrase != "" {
		key, err = ssh.ParsePrivateKeyWithPassphrase(buffer, []byte(s.PassPhrase))
		if err != nil {
			return nil, fmt.Errorf("cannot parse SSH private key file with passphrase, PrivateKey=%s, %w", s.PrivateKey, err)
		}
	} else {
		key, err = ssh.ParsePrivateKey(buffer)
		if err != nil {
			return nil, fmt.Errorf("cannot parse SSH private key file, PrivateKey=%s, %w", s.PrivateKey, err)
		}
	}

	sshConfig := &ssh.ClientConfig{
		User:            s.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(key)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return sshConfig, nil
}

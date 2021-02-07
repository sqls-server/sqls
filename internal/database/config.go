package database

import (
	"fmt"
	"io/ioutil"

	"github.com/lighttiger2505/sqls/dialect"
	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

type Proto string

const (
	ProtoTCP  Proto = "tcp"
	ProtoUDP  Proto = "udp"
	ProtoUnix Proto = "unix"
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
}

type SSHConfig struct {
	Host       string `json:"host" yaml:"host"`
	Port       int    `json:"port" yaml:"port"`
	User       string `json:"user" yaml:"user"`
	PassPhrase string `json:"passPhrase" yaml:"passPhrase"`
	PrivateKey string `json:"privateKey" yaml:"privateKey"`
}

func (s *SSHConfig) Endpoint() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

func (s *SSHConfig) ClientConfig() (*ssh.ClientConfig, error) {
	buffer, err := ioutil.ReadFile(s.PrivateKey)
	if err != nil {
		return nil, xerrors.Errorf("cannot read SSH private key file, PrivateKey=%s, %+v", s.PrivateKey, err)
	}

	var key ssh.Signer
	if s.PassPhrase != "" {
		key, err = ssh.ParsePrivateKeyWithPassphrase(buffer, []byte(s.PassPhrase))
		if err != nil {
			return nil, xerrors.Errorf("cannot parse SSH private key file with passphrase, PrivateKey=%s, %+v", s.PrivateKey, err)
		}
	} else {
		key, err = ssh.ParsePrivateKey(buffer)
		if err != nil {
			return nil, xerrors.Errorf("cannot parse SSH private key file, PrivateKey=%s, %+v", s.PrivateKey, err)
		}
	}

	sshConfig := &ssh.ClientConfig{
		User:            s.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(key)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return sshConfig, nil
}

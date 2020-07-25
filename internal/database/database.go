package database

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

var (
	ErrNotImplementation error = errors.New("not implementation")
)

type Database interface {
	Open() error
	Close() error
	CurrentDatabase(ctx context.Context) (string, error)
	Databases(ctx context.Context) ([]string, error)
	CurrentSchema(ctx context.Context) (string, error)
	Schemas(ctx context.Context) ([]string, error)
	SchemaTables(ctx context.Context) (map[string][]string, error)
	DescribeDatabaseTable(ctx context.Context) ([]*ColumnDesc, error)
	Exec(ctx context.Context, query string) (sql.Result, error)
	Query(ctx context.Context, query string) (*sql.Rows, error)
	SwitchDB(dbName string) error
}

const (
	DefaultMaxIdleConns = 10
	DefaultMaxOpenConns = 5
)

type DBOption struct {
	MaxIdleConns int
	MaxOpenConns int
}

type ColumnDesc struct {
	Schema  string
	Table   string
	Name    string
	Type    string
	Null    string
	Key     string
	Default sql.NullString
	Extra   string
}

func (cd *ColumnDesc) OnelineDesc() string {
	items := []string{}
	if cd.Type != "" {
		items = append(items, cd.Type)
	}
	if cd.Key != "" {
		items = append(items, cd.Key)
	}
	if cd.Extra != "" {
		items = append(items, cd.Extra)
	}
	return strings.Join(items, " ")
}

func (cd *ColumnDesc) OnelineDescWithName() string {
	return fmt.Sprintf("%s: %s", cd.Name, cd.OnelineDesc())
}

func ColumnDoc(tableName string, colDesc *ColumnDesc) string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%s.%s column", tableName, colDesc.Name)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, colDesc.OnelineDesc())
	return buf.String()
}

func TableDoc(tableName string, cols []*ColumnDesc) string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%s table", tableName)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf)
	for _, col := range cols {
		fmt.Fprintf(buf, "- %s", col.OnelineDescWithName())
		fmt.Fprintln(buf)
	}
	return buf.String()
}

type Opener func(*Config) Database

var drivers = make(map[string]Opener)

func Register(name string, f Opener) {
	drivers[name] = f
}

func Open(cfg *Config) (Database, error) {
	d, ok := drivers[cfg.Driver]
	if !ok {
		return nil, xerrors.Errorf("driver not found, %v", cfg.Driver)
	}
	return d(cfg), nil
}

type Proto string

const (
	ProtoTCP  Proto = "tcp"
	ProtoUDP  Proto = "udp"
	ProtoUnix Proto = "unix"
)

type Config struct {
	Alias          string            `json:"alias" yaml:"alias"`
	Driver         string            `json:"driver" yaml:"driver"`
	DataSourceName string            `json:"dataSourceName" yaml:"dataSourceName"`
	Proto          Proto             `json:"proto" yaml:"proto"`
	User           string            `json:"user" yaml:"user"`
	Passwd         string            `json:"passwd" yaml:"passwd"`
	Host           string            `json:"host" yaml:"host"`
	Port           int               `json:"port" yaml:"port"`
	Path           string            `json:"path" yaml:"path"`
	DBName         string            `json:"dbName" yaml:"dbName"`
	Params         map[string]string `json:"params" yaml:"params"`
	SSHCfg         *SSHConfig        `json:"sshConfig" yaml:"sshConfig"`
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

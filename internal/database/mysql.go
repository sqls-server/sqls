package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"strconv"

	mysql "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

type MySQLDB struct {
	Cfg     *Config
	Option  *DBOption
	Conn    *sql.DB
	SSHConn *ssh.Client
	curDB   string
}

func init() {
	Register("mysql", func(cfg *Config) Database {
		return &MySQLDB{
			Cfg:    cfg,
			Option: &DBOption{},
		}
	})
}

func (db *MySQLDB) Open() error {
	cfg, err := genMysqlConfig(db.Cfg)
	if err != nil {
		return err
	}
	if db.curDB != "" {
		cfg.DBName = db.curDB
	}

	if db.Cfg.SSHCfg != nil {
		dbConn, sshConn, err := openOverSSH(cfg.FormatDSN(), db.Cfg.SSHCfg)
		if err != nil {
			return err
		}
		db.Conn = dbConn
		db.SSHConn = sshConn
	} else {
		dbConn, err := sql.Open("mysql", cfg.FormatDSN())
		if err != nil {
			return err
		}
		db.Conn = dbConn
	}
	if err := db.Conn.Ping(); err != nil {
		return err
	}

	db.Conn.SetMaxIdleConns(DefaultMaxIdleConns)
	if db.Option.MaxIdleConns != 0 {
		db.Conn.SetMaxIdleConns(db.Option.MaxIdleConns)
	}
	db.Conn.SetMaxOpenConns(DefaultMaxOpenConns)
	if db.Option.MaxOpenConns != 0 {
		db.Conn.SetMaxOpenConns(db.Option.MaxOpenConns)
	}
	return nil
}

func openOverSSH(dsn string, sshCfg *SSHConfig) (*sql.DB, *ssh.Client, error) {
	sshConfig, err := sshCfg.ClientConfig()
	if err != nil {
		return nil, nil, err
	}
	sshConn, err := ssh.Dial("tcp", sshCfg.Endpoint(), sshConfig)
	if err != nil {
		return nil, nil, xerrors.Errorf("cannot ssh dial, %+v", err)
	}
	mysql.RegisterDialContext("mysql+tcp", (&ViaSSHDialer{sshConn}).Dial)
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, nil, xerrors.Errorf("cannot connect database, %+v", err)
	}
	return conn, sshConn, nil
}

func publicKeyFile(file, passPhrase string) (ssh.AuthMethod, error) {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, xerrors.Errorf("cannot read SSH private key file %s, %s", file, err)
	}

	var key ssh.Signer
	if passPhrase != "" {
		key, err = ssh.ParsePrivateKeyWithPassphrase(buffer, []byte(passPhrase))
		if err != nil {
			return nil, xerrors.Errorf("cannot parse SSH private key file with passphrase, %s, %s", file, err)
		}
	} else {
		key, err = ssh.ParsePrivateKey(buffer)
		if err != nil {
			return nil, xerrors.Errorf("cannot parse SSH private key file, %s, %s", file, err)
		}
	}
	return ssh.PublicKeys(key), nil
}

func (db *MySQLDB) Close() error {
	if err := db.Conn.Close(); err != nil {
		return err
	}
	if db.SSHConn != nil {
		if err := db.SSHConn.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (db *MySQLDB) Databases() ([]string, error) {
	rows, err := db.Conn.Query("SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	databases := []string{}
	for rows.Next() {
		var database string
		if err := rows.Scan(&database); err != nil {
			return nil, err
		}
		databases = append(databases, database)
	}
	return databases, nil
}

func (db *MySQLDB) Tables() ([]string, error) {
	rows, err := db.Conn.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	tables := []string{}
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, nil
}

func (db *MySQLDB) DescribeTable(tableName string) ([]*ColumnDesc, error) {
	rows, err := db.Conn.Query("DESC " + tableName)
	if err != nil {
		return nil, err
	}
	tableInfos := []*ColumnDesc{}
	for rows.Next() {
		var tableInfo ColumnDesc
		err := rows.Scan(
			&tableInfo.Name,
			&tableInfo.Type,
			&tableInfo.Null,
			&tableInfo.Key,
			&tableInfo.Default,
			&tableInfo.Extra,
		)
		if err != nil {
			return nil, err
		}
		tableInfos = append(tableInfos, &tableInfo)
	}
	return tableInfos, nil
}

func (db *MySQLDB) Exec(ctx context.Context, query string) (sql.Result, error) {
	return db.Conn.ExecContext(ctx, query)
}

func (db *MySQLDB) Query(ctx context.Context, query string) (*sql.Rows, error) {
	return db.Conn.QueryContext(ctx, query)
}

func (db *MySQLDB) SwitchDB(dbName string) error {
	db.curDB = dbName
	return nil
}

func genMysqlConfig(connCfg *Config) (*mysql.Config, error) {
	cfg := mysql.NewConfig()

	if connCfg.DataSourceName != "" {
		return mysql.ParseDSN(connCfg.DataSourceName)
	}

	cfg.User = connCfg.User
	cfg.Passwd = connCfg.Passwd
	cfg.DBName = connCfg.DBName

	switch connCfg.Proto {
	case ProtoTCP, ProtoUDP:
		host, port := connCfg.Host, connCfg.Port
		if host == "" {
			host = "127.0.0.1"
		}
		if port == 0 {
			port = 3306
		}
		cfg.Addr = host + ":" + strconv.Itoa(port)
		cfg.Net = string(connCfg.Proto)
	case ProtoUnix:
		if connCfg.Path != "" {
			cfg.Addr = "/tmp/mysql.sock"
			break
		}
		cfg.Addr = connCfg.Path
		cfg.Net = string(connCfg.Proto)
	default:
		return nil, fmt.Errorf("default addr for network %s unknown", connCfg.Proto)
	}

	cfg.Params = connCfg.Params

	return cfg, nil
}

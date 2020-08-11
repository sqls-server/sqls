package database

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

var driverOpeners = make(map[string]ConnOpener)
var driverFactories = make(map[string]RepositoryFactory)

type ConnOpener func(*Config) (*DBConn, error)
type RepositoryFactory func(*sql.DB) DBRepository

type DBConn struct {
	Conn    *sql.DB
	SSHConn *ssh.Client
}

func (db *DBConn) Close() error {
	if db == nil {
		return nil
	}
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

func RegisterConn(name string, opener ConnOpener) {
	if _, ok := driverOpeners[name]; ok {
		panic(fmt.Sprintf("driver %s is already registered", name))
	}
	driverOpeners[name] = opener
}

func RegisterFactory(name string, factory RepositoryFactory) {
	if _, ok := driverFactories[name]; ok {
		panic(fmt.Sprintf("driver %s is already registered", name))
	}
	driverFactories[name] = factory
}

func RegisteredConn(name string) bool {
	_, ok := driverOpeners[name]
	return ok
}

func OpenConn(cfg *Config) (*DBConn, error) {
	fn, ok := driverOpeners[cfg.Driver]
	if !ok {
		return nil, xerrors.Errorf("driver not found, %s", cfg.Driver)
	}
	return fn(cfg)
}

func CreateRepository(driver string, db *sql.DB) (DBRepository, error) {
	fn, ok := driverFactories[driver]
	if !ok {
		return nil, xerrors.Errorf("driver not found, %s", driver)
	}
	return fn(db), nil
}

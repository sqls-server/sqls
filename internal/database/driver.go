package database

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

var driverOpeners = make(map[string]Opener)
var driverFactories = make(map[string]Factory)

type Opener func(*DBConfig) (*DBConnection, error)
type Factory func(*sql.DB) DBRepository

type DBConnection struct {
	Conn    *sql.DB
	SSHConn *ssh.Client
}

func (db *DBConnection) Close() error {
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

func RegisterOpen(name string, opener Opener) {
	if _, ok := driverOpeners[name]; ok {
		panic(fmt.Sprintf("driver open %s method is already registered", name))
	}
	driverOpeners[name] = opener
}

func RegisterFactory(name string, factory Factory) {
	if _, ok := driverFactories[name]; ok {
		panic(fmt.Sprintf("driver factory %s already registered", name))
	}
	driverFactories[name] = factory
}

func Registered(name string) bool {
	_, ok1 := driverOpeners[name]
	_, ok2 := driverFactories[name]
	return ok1 && ok2
}

func Open(cfg *DBConfig) (*DBConnection, error) {
	OpenFn, ok := driverOpeners[cfg.Driver]
	if !ok {
		return nil, xerrors.Errorf("driver not found, %s", cfg.Driver)
	}
	return OpenFn(cfg)
}

func CreateRepository(driver string, db *sql.DB) (DBRepository, error) {
	FactoryFn, ok := driverFactories[driver]
	if !ok {
		return nil, xerrors.Errorf("driver not found, %s", driver)
	}
	return FactoryFn(db), nil
}

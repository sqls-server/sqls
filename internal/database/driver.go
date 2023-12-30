package database

import (
	"database/sql"
	"fmt"

	"github.com/sqls-server/sqls/dialect"
	"golang.org/x/crypto/ssh"
)

var driverOpeners = make(map[dialect.DatabaseDriver]Opener)
var driverFactories = make(map[dialect.DatabaseDriver]Factory)

type Opener func(*DBConfig) (*DBConnection, error)
type Factory func(*sql.DB) DBRepository

type DBConnection struct {
	Conn    *sql.DB
	SSHConn *ssh.Client
	Driver  dialect.DatabaseDriver
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

func RegisterOpen(name dialect.DatabaseDriver, opener Opener) {
	if _, ok := driverOpeners[name]; ok {
		panic(fmt.Sprintf("driver open %s method is already registered", name))
	}
	driverOpeners[name] = opener
}

func RegisterFactory(name dialect.DatabaseDriver, factory Factory) {
	if _, ok := driverFactories[name]; ok {
		panic(fmt.Sprintf("driver factory %s already registered", name))
	}
	driverFactories[name] = factory
}

func Registered(name dialect.DatabaseDriver) bool {
	_, ok1 := driverOpeners[name]
	_, ok2 := driverFactories[name]
	return ok1 && ok2
}

func Open(cfg *DBConfig) (*DBConnection, error) {
	OpenFn, ok := driverOpeners[cfg.Driver]
	if !ok {
		return nil, fmt.Errorf("driver not found, %s", cfg.Driver)
	}
	return OpenFn(cfg)
}

func CreateRepository(driver dialect.DatabaseDriver, db *sql.DB) (DBRepository, error) {
	FactoryFn, ok := driverFactories[driver]
	if !ok {
		return nil, fmt.Errorf("driver not found, %s", driver)
	}
	return FactoryFn(db), nil
}

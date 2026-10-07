package database

import (
	"testing"
)

func TestDBConnection_Close(t *testing.T) {
	t.Run("nil DBConnection", func(t *testing.T) {
		var conn *DBConnection
		if err := conn.Close(); err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("DBConnection with nil Conn does not panic", func(t *testing.T) {
		conn := &DBConnection{
			Conn: nil,
		}
		if err := conn.Close(); err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})
}

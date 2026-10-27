package database

import (
	"context"
	"database/sql/driver"
	"fmt"
	"net"
	"testing"

	"github.com/jfcote87/sshdb"
	mssql "github.com/microsoft/go-mssqldb"
)

var _ sshdb.Driver = mssqlTunnelDriver{}

func Test_genMssqlConfig(t *testing.T) {
	tests := []struct {
		name    string
		connCfg *DBConfig
		want    string
		wantErr bool
	}{
		{
			name: "",
			connCfg: &DBConfig{
				Alias:          "",
				Driver:         "mssql",
				DataSourceName: "",
				Proto:          "tcp",
				User:           "sa",
				Passwd:         "mysecretpassword1234",
				Host:           "127.0.0.1",
				Port:           11433,
				Path:           "",
				DBName:         "dvdrental",
				Params: map[string]string{
					"ApplicationIntent": "ReadOnly",
				},
			},
			want:    "ApplicationIntent=ReadOnly;database=dvdrental;password=mysecretpassword1234;port=11433;server=127.0.0.1;user=sa",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := genMssqlConfig(tt.connCfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("genMssqlConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_mssqlTunnelDriver_Name(t *testing.T) {
	d := mssqlTunnelDriver{}
	if got := d.Name(); got != "mssql" {
		t.Errorf("Name() = %q, want %q", got, "mssql")
	}
}

type stubMssqlDialer struct {
	called bool
}

func (s *stubMssqlDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	s.called = true
	return nil, fmt.Errorf("stubMssqlDialer should not be invoked by OpenConnector")
}

func Test_mssqlTunnelDriver_OpenConnector(t *testing.T) {
	validDSN := "server=127.0.0.1;user=sa;password=secret;database=dvdrental"
	invalidDSN := "log=abc"

	t.Run("success assigns dialer to connector", func(t *testing.T) {
		dialer := &stubMssqlDialer{}
		d := mssqlTunnelDriver{}

		got, err := d.OpenConnector(dialer, validDSN)
		if err != nil {
			t.Fatalf("OpenConnector() unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("OpenConnector() returned nil connector")
		}

		mc, ok := got.(*mssql.Connector)
		if !ok {
			t.Fatalf("OpenConnector() returned %T, want *mssql.Connector", got)
		}
		if mc.Dialer == nil {
			t.Fatal("connector.Dialer is nil, expected the sshdb dialer to be assigned")
		}
		if assigned, ok := mc.Dialer.(*stubMssqlDialer); !ok || assigned != dialer {
			t.Fatalf("connector.Dialer = %p (%T), want the same *stubMssqlDialer instance passed in", mc.Dialer, mc.Dialer)
		}
		if dialer.called {
			t.Fatal("OpenConnector() must not invoke the dialer")
		}
	})

	t.Run("invalid DSN propagates parse error", func(t *testing.T) {
		d := mssqlTunnelDriver{}
		got, err := d.OpenConnector(&stubMssqlDialer{}, invalidDSN)
		if err == nil {
			t.Fatal("OpenConnector() expected error for invalid DSN, got nil")
		}
		if got != nil {
			t.Fatalf("OpenConnector() returned non-nil connector %T on error", got)
		}
	})

	t.Run("returned connector satisfies driver.Connector", func(t *testing.T) {
		got, err := mssqlTunnelDriver{}.OpenConnector(&stubMssqlDialer{}, validDSN)
		if err != nil {
			t.Fatalf("OpenConnector() unexpected error: %v", err)
		}
		var _ driver.Connector = got
	})
}

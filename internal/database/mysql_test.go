package database

import (
	"testing"
)

func Test_genMysqlConfig(t *testing.T) {
	tests := []struct {
		name     string
		connCfg  *DBConfig
		wantNet  string
		wantAddr string
	}{
		{
			name: "tcp defaults",
			connCfg: &DBConfig{
				Proto: ProtoTCP,
				User:  "user",
			},
			wantNet:  "tcp",
			wantAddr: "127.0.0.1:3306",
		},
		{
			name: "unix socket with path",
			connCfg: &DBConfig{
				Proto: ProtoUnix,
				User:  "user",
				Path:  "/var/run/mysqld/mysqld.sock",
			},
			wantNet:  "unix",
			wantAddr: "/var/run/mysqld/mysqld.sock",
		},
		{
			name: "unix socket without path",
			connCfg: &DBConfig{
				Proto: ProtoUnix,
				User:  "user",
			},
			wantNet:  "unix",
			wantAddr: "/tmp/mysql.sock",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := genMysqlConfig(tt.connCfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Net != tt.wantNet {
				t.Errorf("want Net %q, got %q", tt.wantNet, cfg.Net)
			}
			if cfg.Addr != tt.wantAddr {
				t.Errorf("want Addr %q, got %q", tt.wantAddr, cfg.Addr)
			}
		})
	}
}

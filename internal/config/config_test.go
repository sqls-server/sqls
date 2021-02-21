package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lighttiger2505/sqls/internal/database"
)

func TestGetConfig(t *testing.T) {
	type args struct {
		fp string
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		{
			name: "basic",
			args: args{
				fp: "basic.yml",
			},
			want: &Config{
				LowercaseKeywords: true,
				Connections: []*database.DBConfig{
					{
						Alias:  "sqls_mysql",
						Driver: "mysql",
						Proto:  "tcp",
						User:   "root",
						Passwd: "root",
						Host:   "127.0.0.1",
						Port:   13306,
						DBName: "world",
						Params: map[string]string{"autocommit": "true", "tls": "skip-verify"},
					},
					{
						Alias:          "sqls_sqlite3",
						Driver:         "sqlite3",
						DataSourceName: "file:/home/lighttiger2505/chinook.db",
					},
					{
						Alias:  "sqls_postgresql",
						Driver: "postgresql",
						Proto:  "tcp",
						User:   "postgres",
						Passwd: "mysecretpassword1234",
						Host:   "127.0.0.1",
						Port:   15432,
						DBName: "dvdrental",
						Params: map[string]string{"sslmode": "disable"},
					},
					{
						Alias:  "mysql_with_bastion",
						Driver: "mysql",
						Proto:  "tcp",
						User:   "admin",
						Passwd: "Q+ACgv12ABx/",
						Host:   "192.168.121.163",
						Port:   3306,
						DBName: "world",
						SSHCfg: &database.SSHConfig{
							Host:       "192.168.121.168",
							Port:       22,
							User:       "vagrant",
							PassPhrase: "passphrase1234",
							PrivateKey: "/home/lighttiger2505/.ssh/id_rsa",
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		packageDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("cannot get package path, Err=%v", err)
		}
		testFile := filepath.Join(packageDir, "testdata", tt.args.fp)

		t.Run(tt.name, func(t *testing.T) {
			got, err := GetConfig(testFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unmatch (- want, + got):\n%s", diff)
			}
		})
	}
}

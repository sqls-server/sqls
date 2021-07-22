package database

import (
	"testing"

	_ "github.com/jackc/pgx/v4/stdlib"
)

func Test_genPostgresConfig(t *testing.T) {
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
				Driver:         "postgresql",
				DataSourceName: "",
				Proto:          "tcp",
				User:           "postgres",
				Passwd:         "mysecretpassword1234",
				Host:           "127.0.0.1",
				Port:           15432,
				Path:           "",
				DBName:         "dvdrental",
				Params: map[string]string{
					"sslmode": "disable",
				},
			},
			want:    "dbname=dvdrental host=127.0.0.1 password=mysecretpassword1234 port=15432 sslmode=disable user=postgres",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := genPostgresConfig(tt.connCfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("genPostgresConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

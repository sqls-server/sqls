package database

import (
	"testing"

	_ "github.com/microsoft/go-mssqldb"
)

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

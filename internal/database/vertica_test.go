package database

import "testing"

func Test_genVerticaConfig(t *testing.T) {
	tests := []struct {
		name    string
		connCfg *DBConfig
		want    string
	}{
		{
			name: "use datasource name",
			connCfg: &DBConfig{
				DataSourceName: "vertica://user:pwd@localhost:5433/db",
			},
			want: "vertica://user:pwd@localhost:5433/db",
		},
		{
			name: "use config properties",
			connCfg: &DBConfig{
				User:   "dbadmin",
				Passwd: "secure",
				Host:   "example.com",
				Port:   15433,
				DBName: "vdb",
			},
			want: "vertica://dbadmin:secure@example.com:15433/vdb",
		},
		{
			name: "defaults",
			connCfg: &DBConfig{
				User:   "dbadmin",
				Passwd: "secure",
				DBName: "vdb",
			},
			want: "vertica://dbadmin:secure@127.0.0.1:5433/vdb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := genVerticaConfig(tt.connCfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

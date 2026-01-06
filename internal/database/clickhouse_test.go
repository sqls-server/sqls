package database

import "testing"

func TestGenClickhouseDsn(t *testing.T) {
	type testCase struct {
		name    string
		connCfg *DBConfig
		want    string
		wantErr bool
	}

	tests := []testCase{
		{
			name: "use datasource name",
			connCfg: &DBConfig{
				DataSourceName: "clickhouse://user:pwd@localhost:9001",
				Driver:         "clickhouse",
			},
			want:    "clickhouse://user:pwd@localhost:9001",
			wantErr: false,
		},
		{
			name: "use config properties",
			connCfg: &DBConfig{
				Alias:          "",
				DataSourceName: "",
				Driver:         "clickhouse",
				Proto:          "tcp",
				User:           "test",
				Passwd:         "secure",
				Host:           "localhost",
				Port:           9001,
				Path:           "",
				DBName:         "default",
				Params: map[string]string{
					"dial_timeout": "200ms",
				},
			},
			want: "clickhouse://test:secure@localhost:9001/default?dial_timeout=200ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := genClickhouseDsn(tt.connCfg)

			if (err != nil) != tt.wantErr {
				t.Errorf("genClickhouseDsn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})

	}
}

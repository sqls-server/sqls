package database

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/godror/godror"
)

func Test_genOracleDummy(t *testing.T) {
	db, err1 := sql.Open("godror", `user="SYSTEM" password="P1ssword" connectString="localhost:1521/XE"`)
	if err1 != nil {
		t.Errorf(err1.Error())
	}

	rows, err := db.Query("select * from all_tables")
	if err != nil {
		t.Errorf(err.Error())
	}

	cols, err := rows.Columns()
	if err != nil {
		fmt.Println("Failed to get columns", err)
		return
	}

	// Result is your slice string.
	rawResult := make([][]byte, len(cols))
	result := make([]string, len(cols))
	dest := make([]interface{}, len(cols)) // A temporary interface{} slice
	for i, _ := range rawResult {
		dest[i] = &rawResult[i] // Put pointers to each string in the interface slice
	}

	for rows.Next() {
		err = rows.Scan(dest...)
		if err != nil {
			fmt.Println("Failed to scan row", err)
			return
		}

		for i, raw := range rawResult {
			if raw == nil {
				result[i] = "\\N"
			} else {
				result[i] = string(raw)
			}
		}

		t.Error(result)
	}
	return
}

func Test_genOracleConfig(t *testing.T) {
	tests := []struct {
		name    string
		connCfg *DBConfig
		want    string
		wantErr bool
	}{
		{
			name: "",
			connCfg: &DBConfig{
				Alias:          "TestDB",
				Driver:         "oracle",
				DataSourceName: "XE",
				Proto:          "tcp",
				User:           "SYSTEM",
				Passwd:         "P1ssword",
				Host:           "127.0.0.1",
				Port:           1521,
				Path:           "",
				DBName:         "XE",
			},
			want:    "XE",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := genOracleConfig(tt.connCfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("genOracleConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

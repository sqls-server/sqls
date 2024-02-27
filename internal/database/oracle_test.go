package database

// import (
// 	"context"
// 	"database/sql"
// 	"fmt"
// 	"testing"
//
// 	_ "github.com/godror/godror"
// )
//
// // 	Generic Testing template for multiple DB driver
// //  Test_genDriver: Test if driver config correctly
// //  Test_ConnectByConfig: Test if Connection setup by config successful
// //  Test_BasicOperation: Test all function except DescribeDatabaseTableBySchema
// //  Test_SchemaTableOperation: Test DescribeDatabaseTableBySchema
//
// func Test_genDriver(t *testing.T) {
// 	//db, err1 := sql.Open("godror", `user="SYSTEM" password="P1ssword" connectString="localhost:1521/XE"`)
// 	db, err1 := sql.Open("godror", `SYSTEM/P1ssword@localhost:1521/XE`)
// 	if err1 != nil {
// 		t.Errorf(err1.Error())
// 	}
//
// 	rows, err := db.Query("SELECT SYS_CONTEXT('USERENV','INSTANCE_NAME') FROM DUAL")
// 	if err != nil {
// 		t.Errorf(err.Error())
// 	}
//
// 	cols, err := rows.Columns()
// 	if err != nil {
// 		fmt.Println("Failed to get columns", err)
// 		return
// 	}
//
// 	// Result is your slice string.
// 	rawResult := make([][]byte, len(cols))
// 	result := make([]string, len(cols))
// 	dest := make([]interface{}, len(cols)) // A temporary interface{} slice
// 	for i, _ := range rawResult {
// 		dest[i] = &rawResult[i] // Put pointers to each string in the interface slice
// 	}
//
// 	for rows.Next() {
// 		err = rows.Scan(dest...)
// 		if err != nil {
// 			fmt.Println("Failed to scan row", err)
// 			return
// 		}
//
// 		for i, raw := range rawResult {
// 			if raw == nil {
// 				result[i] = "\\N"
// 			} else {
// 				result[i] = string(raw)
// 			}
// 		}
//
// 		t.Error(result)
// 	}
// }
//
// func Test_ConnectByConfig(t *testing.T) {
// 	// Suit all DB
// 	tests := []struct {
// 		name    string
// 		connCfg *DBConfig
// 		want    string
// 		wantErr bool
// 		ctx     context.Context
// 	}{
// 		/*	{
// 			name: "test1",
// 			connCfg: &DBConfig{
// 				Alias:  "TestDB",
// 				Driver: "oracle",
// 				Proto:  "tcp",
// 				User:   "SYSTEM",
// 				Passwd: "P1ssword",
// 				Host:   "127.0.0.1",
// 				Port:   1521,
// 				Path:   "",
// 				DBName: "XE",
// 			},
// 			want:    "XE",
// 			wantErr: false,
// 		},*/
// 		{
// 			name: "test2",
// 			connCfg: &DBConfig{
// 				Alias:          "TestDB",
// 				Driver:         "oracle",
// 				DataSourceName: "SYSTEM/P1ssword@localhost:1521/XE",
// 			},
// 			want:    "XE",
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		tt.ctx = context.Background()
// 		t.Run(tt.name, func(t *testing.T) {
// 			// Connect to DB
// 			db, err := Open(tt.connCfg)
// 			if err != nil {
// 				t.Errorf("genOracleConfig() error = %v", err)
// 				return
// 			}
// 			repo, err := CreateRepository(tt.connCfg.Driver, db.Conn)
// 			if err != nil {
// 				t.Errorf("NewOracleDBRepository() error = %v", err)
// 				return
// 			}
// 			// End Connect to DB
//
// 			pt, err := repo.CurrentDatabase(tt.ctx)
// 			if err != nil {
// 				t.Log(pt)
// 				t.Errorf("NewOracleDBRepository() error = %v", err)
// 				return
// 			}
// 		},
// 		)
// 	}
// }
//
// func Test_BasicOperation(t *testing.T) {
// 	tests := []struct {
// 		name    string
// 		connCfg *DBConfig
// 		want    string
// 		wantErr bool
// 		ctx     context.Context
// 	}{
// 		/*	{
// 			name: "test1",
// 			connCfg: &DBConfig{
// 				Alias:  "TestDB",
// 				Driver: "oracle",
// 				Proto:  "tcp",
// 				User:   "SYSTEM",
// 				Passwd: "P1ssword",
// 				Host:   "127.0.0.1",
// 				Port:   1521,
// 				Path:   "",
// 				DBName: "XE",
// 			},
// 			want:    "XE",
// 			wantErr: false,
// 		},*/
// 		{
// 			name: "test2",
// 			connCfg: &DBConfig{
// 				Alias:          "TestDB",
// 				Driver:         "oracle",
// 				DataSourceName: "SYSTEM/P1ssword@localhost:1521/XE",
// 			},
// 			want:    "XE",
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		tt.ctx = context.Background()
// 		t.Run(tt.name, func(t *testing.T) {
// 			// Connect to DB
// 			db, err := Open(tt.connCfg)
//
// 			if err != nil {
// 				t.Errorf("genOracleConfig() error = %v", err)
// 				return
// 			}
// 			repo, err := CreateRepository(tt.connCfg.Driver, db.Conn)
// 			if err != nil {
// 				t.Errorf("NewOracleDBRepository() error = %v", err)
// 				return
// 			}
// 			// End Connect to DB
//
// 			pt, err := repo.CurrentDatabase(tt.ctx)
// 			if err != nil {
// 				t.Log(pt)
// 				t.Errorf("NewOracleDBRepository() error = %v", err)
// 				return
// 			}
// 			pt, err = repo.CurrentSchema(tt.ctx)
// 			if err != nil {
// 				t.Log(pt)
// 				t.Errorf("NewOracleDBRepository() error = %v", err)
// 				return
// 			}
// 			_, err = repo.SchemaTables(tt.ctx)
// 			if err != nil {
// 				t.Log(pt)
// 				t.Errorf("NewOracleDBRepository() error = %v", err)
// 				return
// 			}
// 			_, err = repo.Databases(tt.ctx)
// 			if err != nil {
// 				t.Errorf("NewOracleDBRepository() error = %v", err)
// 				return
// 			}
// 			_, err = repo.DescribeDatabaseTable(tt.ctx)
// 			if err != nil {
// 				t.Errorf("NewOracleDBRepository() error = %v", err)
// 				return
// 			}
// 			tt.ctx.Done()
// 			schemalist, err := repo.Schemas(tt.ctx)
// 			if err != nil {
// 				t.Errorf("NewOracleDBRepository() error = %v", err)
// 				return
// 			}
// 			tt.ctx.Done()
// 			for _, sch := range schemalist {
// 				t.Errorf("sch %v", sch)
// 				repo.DescribeDatabaseTableBySchema(tt.ctx, sch)
// 				if err != nil {
// 					t.Errorf("NewOracleDBRepository() error = %v", err.Error())
// 					t.Fatal()
// 					return
// 				}
// 				tt.ctx.Done()
// 			}
// 			query := "SELECT USERNAME FROM SYS.ALL_USERS ORDER BY USERNAME"
// 			repo.Exec(tt.ctx, query)
// 			repo.Query(tt.ctx, query)
//
// 		})
// 	}
// }
//
// func Test_SchemaTableOperation(t *testing.T) {
// 	// Suit all DB
// 	tests := []struct {
// 		name    string
// 		connCfg *DBConfig
// 		want    string
// 		wantErr bool
// 		ctx     context.Context
// 	}{
// 		/*	{
// 			name: "test1",
// 			connCfg: &DBConfig{
// 				Alias:  "TestDB",
// 				Driver: "oracle",
// 				Proto:  "tcp",
// 				User:   "SYSTEM",
// 				Passwd: "P1ssword",
// 				Host:   "127.0.0.1",
// 				Port:   1521,
// 				Path:   "",
// 				DBName: "XE",
// 			},
// 			want:    "XE",
// 			wantErr: false,
// 		},*/
// 		{
// 			name: "test2",
// 			connCfg: &DBConfig{
// 				Alias:          "TestDB",
// 				Driver:         "oracle",
// 				DataSourceName: "SYSTEM/P1ssword@localhost:1521/XE",
// 			},
// 			want:    "XE",
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		tt.ctx = context.Background()
// 		t.Run(tt.name, func(t *testing.T) {
// 			// Connect to DB
// 			db, err := Open(tt.connCfg)
//
// 			if err != nil {
// 				t.Errorf("genOracleConfig() error = %v", err)
// 				return
// 			}
// 			repo, err := CreateRepository(tt.connCfg.Driver, db.Conn)
// 			if err != nil {
// 				t.Errorf("NewOracleDBRepository() error = %v", err)
// 				return
// 			}
// 			// End Connect to DB
//
// 			schemalist, err := repo.Schemas(tt.ctx)
// 			if err != nil {
// 				t.Errorf("NewOracleDBRepository() error = %v", err)
// 				return
// 			}
// 			tt.ctx.Done()
// 			for _, sch := range schemalist {
// 				t.Logf("sch %v", sch)
// 				repo.DescribeDatabaseTableBySchema(tt.ctx, sch)
// 				if err != nil {
// 					t.Errorf("NewOracleDBRepository() error = %v", err.Error())
// 					t.Fatal()
// 					return
// 				}
// 				tt.ctx.Done()
// 			}
// 		},
// 		)
// 	}
// }

package database

import "testing"

func TestQueryExecType(t *testing.T) {
	type args struct {
		prefix string
		sqlstr string
	}
	tests := []struct {
		name         string
		prefix       string
		sqlstr       string
		wantPrefix   string
		wantExecType bool
	}{
		{
			name:         "",
			prefix:       "select * from city",
			sqlstr:       "",
			wantPrefix:   "SELECT",
			wantExecType: true,
		},
		{
			name:         "",
			prefix:       "explain select * from city",
			sqlstr:       "",
			wantPrefix:   "EXPLAIN",
			wantExecType: true,
		},
		{
			name:         "",
			prefix:       "insert into city values (8181, 'Kabul', 'AFG', 'Kabol', 1780000);",
			sqlstr:       "",
			wantPrefix:   "INSERT",
			wantExecType: false,
		},
		{
			name:         "",
			prefix:       "delete from city where id = 8181;",
			sqlstr:       "",
			wantPrefix:   "DELETE",
			wantExecType: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, execType := QueryExecType(tt.prefix, tt.sqlstr)
			if prefix != tt.wantPrefix {
				t.Errorf("QueryExecType() got = %v, want %v", prefix, tt.wantPrefix)
			}
			if execType != tt.wantExecType {
				t.Errorf("QueryExecType() got1 = %v, want %v", execType, tt.wantExecType)
			}
		})
	}
}

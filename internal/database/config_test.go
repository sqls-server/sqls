package database

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestResolvePassword(t *testing.T) {
	type testCase struct {
		title    string
		dbConfig DBConfig
		want     string
		wantErr  bool
	}

	testCases := []testCase{
		{
			title: "only password",
			dbConfig: DBConfig{
				Passwd: "test",
			},
			want: "test",
		},
		{
			title: "only command",
			dbConfig: DBConfig{
				PasswdCmd: []string{"echo", "-n", "secure"},
			},
			want: "secure",
		},
		{
			title: "password and command",
			dbConfig: DBConfig{
				Passwd:    "test",
				PasswdCmd: []string{"echo", "-n", "secure"},
			},
			want: "secure",
		},
		{
			title: "failing command",
			dbConfig: DBConfig{
				Passwd:    "test",
				PasswdCmd: []string{"false"},
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.title, func(t *testing.T) {
			got, err := tt.dbConfig.ResolvePassword()
			if err != nil {
				if !tt.wantErr {
					t.Errorf("ResolvePassword() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unmatch (- want, + got):\n%s", diff)
			}
		})
	}
}

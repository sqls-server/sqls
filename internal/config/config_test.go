package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sqls-server/sqls/internal/database"
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
		errMsg  string
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
						DataSourceName: "file:/home/sqls-server/chinook.db",
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
							PrivateKey: "/home/sqls-server/.ssh/id_rsa",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no driver",
			args: args{
				fp: "no_driver.yml",
			},
			want:    nil,
			wantErr: true,
			errMsg:  "failed validation, required: connections[].driver",
		},
		{
			name: "no connection",
			args: args{
				fp: "no_connection.yml",
			},
			want:    nil,
			wantErr: true,
			errMsg:  "failed validation, required: connections[].dataSourceName or connections[].proto",
		},
		{
			name: "no user",
			args: args{
				fp: "no_user.yml",
			},
			want:    nil,
			wantErr: true,
			errMsg:  "failed validation, required: connections[].user",
		},
		{
			name: "invalid proto",
			args: args{
				fp: "invalid_proto.yml",
			},
			want:    nil,
			wantErr: true,
			errMsg:  "failed validation, invalid: connections[].proto",
		},
		{
			name: "no path",
			args: args{
				fp: "no_path.yml",
			},
			want:    nil,
			wantErr: true,
			errMsg:  "failed validation, required: connections[].path",
		},
		{
			name: "no dsn",
			args: args{
				fp: "no_dsn.yml",
			},
			want: &Config{
				Connections: []*database.DBConfig{
					{
						Alias:          "sqls_sqlite3",
						Driver:         "sqlite3",
						DataSourceName: "",
					},
				},
			},
			wantErr: true,
			errMsg:  "failed validation, required: connections[].dataSourceName",
		},
		{
			name: "no ssh host",
			args: args{
				fp: "no_ssh_host.yml",
			},
			want:    nil,
			wantErr: true,
			errMsg:  "failed validation, required: connections[]sshConfig.host",
		},
		{
			name: "no ssh user",
			args: args{
				fp: "no_ssh_user.yml",
			},
			want:    nil,
			wantErr: true,
			errMsg:  "failed validation, required: connections[].sshConfig.user",
		},
		{
			name: "no ssh private key",
			args: args{
				fp: "no_ssh_private_key.yml",
			},
			want:    nil,
			wantErr: true,
			errMsg:  "failed validation, required: connections[].sshConfig.privateKey",
		},
		{
			name: "oracle config",
			args: args{
				fp: "oracle.yaml",
			},
			want: &Config{
				Connections: []*database.DBConfig{
					{
						Alias:          "TestDB",
						Driver:         "oracle",
						DataSourceName: "SYSTEM/P1ssword@localhost:1521/XE",
					},
				},
			},
			wantErr: true,
			errMsg:  "failed validation, required: connections[].sshConfig.privateKey",
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
			if err != nil {
				if tt.wantErr {
					if err.Error() != tt.errMsg {
						t.Errorf("unmatch error message, want:%q got:%q", tt.errMsg, err.Error())
					}
				} else {
					t.Errorf("GetConfig() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unmatch (- want, + got):\n%s", diff)
			}
		})
	}
}

func TestFindWorkspaceConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Initially empty
	if fp := FindWorkspaceConfigPath(tmpDir); fp != "" {
		t.Fatalf("expected empty, got %s", fp)
	}

	// 2. Add .config/sqls/config.yml
	cfgDir := filepath.Join(tmpDir, ".config", "sqls")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	nestedPath := filepath.Join(cfgDir, "config.yml")
	if err := os.WriteFile(nestedPath, []byte("lowercaseKeywords: true\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if fp := FindWorkspaceConfigPath(tmpDir); fp != nestedPath {
		t.Fatalf("expected %s, got %s", nestedPath, fp)
	}

	// 3. Add sqls.yaml (should take priority over .config/sqls/config.yml)
	sqlsYamlPath := filepath.Join(tmpDir, "sqls.yaml")
	if err := os.WriteFile(sqlsYamlPath, []byte("lowercaseKeywords: false\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if fp := FindWorkspaceConfigPath(tmpDir); fp != sqlsYamlPath {
		t.Fatalf("expected %s, got %s", sqlsYamlPath, fp)
	}

	// 4. Add .sqls.yml (should take top priority)
	dotSqlsYmlPath := filepath.Join(tmpDir, ".sqls.yml")
	if err := os.WriteFile(dotSqlsYmlPath, []byte("lowercaseKeywords: true\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if fp := FindWorkspaceConfigPath(tmpDir); fp != dotSqlsYmlPath {
		t.Fatalf("expected %s, got %s", dotSqlsYmlPath, fp)
	}

	// 5. Test GetWorkspaceConfig
	cfg, err := GetWorkspaceConfig(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.LowercaseKeywords {
		t.Fatalf("expected LowercaseKeywords true")
	}
}

func TestFindDefaultConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Initially neither exists
	if fp := FindDefaultConfigPath(); fp != "" {
		t.Fatalf("expected empty, got %s", fp)
	}

	sqlsDir := filepath.Join(tmpDir, "sqls")
	if err := os.MkdirAll(sqlsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config.yaml
	yamlPath := filepath.Join(sqlsDir, "config.yaml")
	if err := os.WriteFile(yamlPath, []byte("lowercaseKeywords: true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if fp := FindDefaultConfigPath(); fp != yamlPath {
		t.Fatalf("expected %s, got %s", yamlPath, fp)
	}

	// Create config.yml (takes precedence over config.yaml)
	ymlPath := filepath.Join(sqlsDir, "config.yml")
	if err := os.WriteFile(ymlPath, []byte("lowercaseKeywords: false\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if fp := FindDefaultConfigPath(); fp != ymlPath {
		t.Fatalf("expected %s, got %s", ymlPath, fp)
	}
}

func TestURItoPath(t *testing.T) {
	tests := []struct {
		uri  string
		want string
	}{
		{uri: "", want: ""},
		{uri: "/local/path", want: "/local/path"},
		{uri: "file:///home/user/project", want: "/home/user/project"},
		{uri: "file:///var/data", want: "/var/data"},
	}
	for _, tt := range tests {
		got := URItoPath(tt.uri)
		if got != tt.want {
			t.Errorf("URItoPath(%q) = %q, want %q", tt.uri, got, tt.want)
		}
	}
}

func TestGetConfigTOML(t *testing.T) {
	tmpDir := t.TempDir()
	tomlPath := filepath.Join(tmpDir, "config.toml")
	tomlData := `lowercaseKeywords = true

[[connections]]
alias = "my_pg"
driver = "postgresql"
proto = "tcp"
user = "postgres"
passwd = "secretpassword"
host = "localhost"
port = 5432
dbName = "testdb"

[connections.params]
sslmode = "disable"

[connections.sshConfig]
host = "ssh.example.com"
port = 2222
user = "sshuser"
privateKey = "/tmp/id_rsa"
`
	if err := os.WriteFile(tomlPath, []byte(tomlData), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := GetConfig(tomlPath)
	if err != nil {
		t.Fatalf("GetConfig() unexpected error: %v", err)
	}

	want := &Config{
		LowercaseKeywords: true,
		Connections: []*database.DBConfig{
			{
				Alias:  "my_pg",
				Driver: "postgresql",
				Proto:  "tcp",
				User:   "postgres",
				Passwd: "secretpassword",
				Host:   "localhost",
				Port:   5432,
				DBName: "testdb",
				Params: map[string]string{
					"sslmode": "disable",
				},
				SSHCfg: &database.SSHConfig{
					Host:       "ssh.example.com",
					Port:       2222,
					User:       "sshuser",
					PrivateKey: "/tmp/id_rsa",
				},
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unmatch (- want, + got):\n%s", diff)
	}
}

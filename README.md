# sqls: SQL Language Server

![test](https://github.com/sqls-server/sqls/workflows/test/badge.svg)

An implementation of the Language Server Protocol for SQL.

## Note

This project is currently under development and there is no stable release. Therefore, destructive interface changes and configuration changes are expected.

## Features

sqls aims to provide advanced intelligence for you to edit sql in your own editor.

### Support RDBMS

- MySQL([Go-MySQL-Driver](https://github.com/go-sql-driver/mysql))
- PostgreSQL([pgx](https://github.com/jackc/pgx))
- SQLite3([go-sqlite3](https://github.com/mattn/go-sqlite3))
- MSSQL([go-mssqldb](https://github.com/denisenkom/go-mssqldb))
- H2([pgx](https://github.com/CodinGame/h2go))
- Vertica([vertica-sql-go](https://github.com/vertica/vertica-sql-go))

### Language Server Features

#### Auto Completion

![completion](./imgs/sqls-completion.gif)

- DML(Data Manipulation Language)
    - [x] SELECT
        - [x] Sub Query
    - [x] INSERT
    - [x] UPDATE
    - [x] DELETE
- DDL(Data Definition Language)
    - [ ] CREATE TABLE
    - [ ] ALTER TABLE

#### Join completion
If the tables are connected with a foreign key sqls can complete ```JOIN``` statements

![join_completion](imgs/sqls-fk_joins.gif)

#### CodeAction

![code_actions](https://github.com/sqls-server/sqls.vim/blob/master/imgs/sqls_vim_demo.gif)

- [x] Execute SQL
- [ ] Explain SQL
- [x] Switch Connection(Selected Database Connection)
- [x] Switch Database

#### Hover

![hover](./imgs/sqls_hover.gif)

#### Signature Help

![signature_help](./imgs/sqls_signature_help.gif)

#### Document Formatting

![document_format](./imgs/sqls_document_format.gif)

## Installation

```shell
go install github.com/sqls-server/sqls@latest
```

## Editor Plugins

- [sqls.vim](https://github.com/sqls-server/sqls.vim)
- [vscode-sqls](https://github.com/lighttiger2505/vscode-sqls)
- [sqls.nvim](https://github.com/nanotee/sqls.nvim)
- [Emacs LSP mode](https://emacs-lsp.github.io/lsp-mode/page/lsp-sqls/)

## DB Configuration

The connection to the RDBMS is essential to take advantage of the functionality provided by `sqls`.
You need to set the connection to the RDBMS.

### Configuration Methods

There are the following methods for RDBMS connection settings, and they are prioritized in order from the top.
Whichever method you choose, the settings you make will remain the same.

1. Configuration file specified by the `-config` flag
1. `workspace/configuration` set to LSP client
1. Configuration file located in the following location
    - `$XDG_CONFIG_HOME`/sqls/config.yml ("`$HOME`/.config" is used instead of `$XDG_CONFIG_HOME` if it's not set)

### Configuration file sample

```yaml
# Set to true to use lowercase keywords instead of uppercase.
lowercaseKeywords: false
connections:
  - alias: dsn_mysql
    driver: mysql
    dataSourceName: root:root@tcp(127.0.0.1:13306)/world
  - alias: individual_mysql
    driver: mysql
    proto: tcp
    user: root
    passwd: root
    host: 127.0.0.1
    port: 13306
    dbName: world
    params:
      autocommit: "true"
      tls: skip-verify
  - alias: mysql_via_ssh
    driver: mysql
    proto: tcp
    user: admin
    passwd: Q+ACgv12ABx/
    host: 192.168.121.163
    port: 3306
    dbName: world
    sshConfig:
      host: 192.168.121.168
      port: 22
      user: sshuser
      passPhrase: ssspass
      privateKey: /home/sqls-server/.ssh/id_rsa
  - alias: dsn_vertica
    driver: vertica
    dataSourceName: vertica://user:pass@host:5433/dbname
```

### Workspace configuration Sample

- setting example with vim-lsp.

```vim
if executable('sqls')
    augroup LspSqls
        autocmd!
        autocmd User lsp_setup call lsp#register_server({
        \   'name': 'sqls',
        \   'cmd': {server_info->['sqls']},
        \   'whitelist': ['sql'],
        \   'workspace_config': {
        \     'sqls': {
        \       'connections': [
        \         {
        \           'driver': 'mysql',
        \           'dataSourceName': 'root:root@tcp(127.0.0.1:13306)/world',
        \         },
        \         {
        \           'driver': 'postgresql',
        \           'dataSourceName': 'host=127.0.0.1 port=15432 user=postgres password=mysecretpassword1234 dbname=dvdrental sslmode=disable',
        \         },
        \       ],
        \     },
        \   },
        \ })
    augroup END
endif
```

- setting example with coc.nvim.

In `coc-settings.json` opened by `:CocConfig`

```json
{
    "languageserver": {
        "sql": {
            "command": "sqls",
            "args": ["-config", "$HOME/.config/sqls/config.yml"],
            "filetypes": ["sql"],
            "shell": true
        }
    }
}
```

- setting example with [nvim-lspconfig](https://github.com/neovim/nvim-lspconfig/blob/master/doc/server_configurations.md#sqls).

```lua
require'lspconfig'.sqls.setup{
  on_attach = function(client, bufnr)
    require('sqls').on_attach(client, bufnr) -- require sqls.nvim
  end
  settings = {
    sqls = {
      connections = {
        {
          driver = 'mysql',
          dataSourceName = 'root:root@tcp(127.0.0.1:13306)/world',
        },
        {
          driver = 'postgresql',
          dataSourceName = 'host=127.0.0.1 port=15432 user=postgres password=mysecretpassword1234 dbname=dvdrental sslmode=disable',
        },
      },
    },
  },
}
```

- Setting example for Sublime Text 4

  Install the LSP Client by Opening the command palette and run ```Package Control: Install Package```, then select ```LSP```.

  Open ```Preferences > Package Settings > LSP > Settings``` and add the ```"sqls"``` client configuration to the ```"clients"```:
```
{
    "show_diagnostics_count_in_view_status": true,
    "clients": {
        "sqls": {
            "enabled": true,
            "command": ["/path/to/sqls binary"],
            "selector": "source.sql"
        }
    }
}
```

**I'm sorry. Please wait a little longer for other editor settings.**

### Configuration Parameters

The first setting in `connections` is the default connection.

| Key         | Description          |
| ----------- | -------------------- |
| connections | Database connections |

### connections

`dataSourceName` takes precedence over the value set in `proto`, `user`, `passwd`, `host`, `port`, `dbName`, `params`.

| Key            | Description                                 |
| -------------- | ------------------------------------------- |
| alias          | Connection alias name. Optional.            |
| driver         | `mysql`, `postgresql`, `sqlite3`, `mssql`, `h2`. Required. |
| dataSourceName | Data source name.                           |
| proto          | `tcp`, `udp`, `unix`.                       |
| user           | User name                                   |
| passwd         | Password                                    |
| host           | Host                                        |
| port           | Port                                        |
| path           | unix socket path                            |
| dbName         | Database name                               |
| params         | Option params. Optional.                    |
| sshConfig      | ssh config. Optional.                       |

#### sshConfig

| Key        | Description                 |
| ---------- | --------------------------- |
| host       | ssh host. Required.         |
| port       | ssh port. Required.         |
| user       | ssh user. Optional.         |
| privateKey | private key path. Required. |
| passPhrase | passPhrase. Optional.       |

#### DSN (Data Source Name)

See also.

- <https://github.com/go-sql-driver/mysql#dsn-data-source-name>
- <https://pkg.go.dev/github.com/jackc/pgx/v4>
- <https://github.com/mattn/go-sqlite3#connection-string>

## Contributors

This project exists thanks to all the people who contribute.
<a href="https://github.com/sqls-server/sqls/graphs/contributors">
    <img src="https://contrib.rocks/image?repo=sqls-server/sqls" />
</a>

## Inspired

I created sqls inspired by the following OSS.

- [dbcli Tools](https://github.com/dbcli)
    - [mycli](https://www.mycli.net/)
    - [pgcli](https://www.pgcli.com/)
    - [litecli](https://litecli.com/)
- non-validating SQL parser
    - [sqlparse](https://github.com/andialbrecht/sqlparse)

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/lighttiger2505/sqls/internal/database"
	"github.com/lighttiger2505/sqls/internal/lsp"
	"github.com/olekukonko/tablewriter"
	"github.com/sourcegraph/jsonrpc2"
)

const (
	CommandExecuteQuery     = "executeQuery"
	CommandShowDatabases    = "showDatabases"
	CommandShowConnections  = "showConnections"
	CommandSwitchDatabase   = "switchDatabase"
	CommandSwitchConnection = "switchConnections"
)

func (h *Server) handleTextDocumentCodeAction(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.CodeActionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	commands := []lsp.Command{
		{
			Title:     "Execute Query",
			Command:   CommandExecuteQuery,
			Arguments: []interface{}{params.TextDocument.URI},
		},
		{
			Title:     "Show Databases",
			Command:   CommandShowDatabases,
			Arguments: []interface{}{},
		},
		{
			Title:     "Show Connections",
			Command:   CommandShowConnections,
			Arguments: []interface{}{},
		},
		{
			Title:     "Switch Database",
			Command:   CommandSwitchDatabase,
			Arguments: []interface{}{},
		},
		{
			Title:     "Switch Connections",
			Command:   CommandSwitchConnection,
			Arguments: []interface{}{},
		},
	}
	return commands, nil
}

func (s *Server) handleWorkspaceExecuteCommand(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.ExecuteCommandParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	switch params.Command {
	case CommandExecuteQuery:
		return s.executeQuery(params)
	case CommandShowDatabases:
		return s.showDatabases(params)
	case CommandShowConnections:
		return s.showConnections(params)
	case CommandSwitchDatabase:
		return s.switchDatabase(params)
	case CommandSwitchConnection:
		return s.switchConnections(params)
	}
	return nil, fmt.Errorf("unsupported command: %v", params.Command)
}

func (s *Server) executeQuery(params lsp.ExecuteCommandParams) (result interface{}, err error) {
	if s.db == nil {
		return nil, errors.New("connection is closed")
	}
	if len(params.Arguments) != 1 {
		return nil, fmt.Errorf("required arguments were not provided: <File URI>")
	}
	uri, ok := params.Arguments[0].(string)
	if !ok {
		return nil, fmt.Errorf("specify the file uri as a string")
	}
	f, ok := s.files[uri]
	if !ok {
		return nil, fmt.Errorf("document not found, %q", uri)
	}

	if err := s.db.Open(); err != nil {
		return nil, err
	}
	defer s.db.Close()

	if _, isQuery := database.QueryExecType(f.Text, ""); isQuery {
		rows, err := s.db.Query(context.Background(), f.Text)
		if err != nil {
			return err.Error(), nil
		}
		columns, err := database.Columns(rows)
		if err != nil {
			return nil, err
		}
		stringRows, err := database.ScanRows(rows, len(columns))
		if err != nil {
			return nil, err
		}

		buf := new(bytes.Buffer)
		table := tablewriter.NewWriter(buf)
		table.SetHeader(columns)
		for _, stringRow := range stringRows {
			table.Append(stringRow)
		}
		table.Render()
		return buf.String() + fmt.Sprintf("%d rows in set", len(stringRows)), nil
	} else {
		result, err := s.db.Exec(context.Background(), f.Text)
		if err != nil {
			return err.Error(), nil
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}
		return fmt.Sprintf("Query OK, %d row affected", rowsAffected), nil
	}
}

func (s *Server) showDatabases(params lsp.ExecuteCommandParams) (result interface{}, err error) {
	if err := s.db.Open(); err != nil {
		return nil, err
	}
	databases, err := s.db.Databases()
	if err != nil {
		return nil, err
	}
	defer s.db.Close()
	return strings.Join(databases, "\n"), nil
}

func (s *Server) switchDatabase(params lsp.ExecuteCommandParams) (result interface{}, err error) {
	if len(params.Arguments) != 1 {
		return nil, fmt.Errorf("required arguments were not provided: <DB Name>")
	}
	dbName, ok := params.Arguments[0].(string)
	if !ok {
		return nil, fmt.Errorf("specify the db name as a string")
	}

	// Reconnect database
	s.curDBName = dbName
	if err := s.ConnectDatabase(); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *Server) showConnections(params lsp.ExecuteCommandParams) (result interface{}, err error) {
	results := []string{}
	conns := s.getConfig().Connections
	for i, conn := range conns {
		var desc string
		if conn.DataSourceName != "" {
			desc = conn.DataSourceName
		} else {
			switch conn.Proto {
			case database.ProtoTCP:
				desc = fmt.Sprintf("tcp(%s:%d)/%s", conn.Host, conn.Port, conn.DBName)
			case database.ProtoUnix:
				desc = fmt.Sprintf("unix(%s)/%s", conn.Path, conn.DBName)
			}
		}
		res := fmt.Sprintf("%d %s %s %s", i+1, conn.Driver, conn.Alias, desc)
		results = append(results, res)
	}
	return strings.Join(results, "\n"), nil
}

func (s *Server) switchConnections(params lsp.ExecuteCommandParams) (result interface{}, err error) {
	if len(params.Arguments) != 1 {
		return nil, fmt.Errorf("required arguments were not provided: <Connection Index>")
	}
	indexStr, ok := params.Arguments[0].(string)
	if !ok {
		return nil, fmt.Errorf("specify the connection index as a number")
	}
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return nil, fmt.Errorf("specify the connection index as a number, %s", err)
	}
	index = index - 1

	// Reconnect database
	s.curConnectionIndex = index
	if err := s.ConnectDatabase(); err != nil {
		return nil, err
	}
	return nil, nil
}

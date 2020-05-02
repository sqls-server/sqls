package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/internal/database"
	"github.com/lighttiger2505/sqls/internal/lsp"
	"github.com/lighttiger2505/sqls/parser"
	"github.com/olekukonko/tablewriter"
	"github.com/sourcegraph/jsonrpc2"
	"golang.org/x/xerrors"
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
		return nil, errors.New("database connection is not open")
	}
	if len(params.Arguments) == 0 {
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

	showVertical := false
	if len(params.Arguments) > 1 {
		showVerticalFlag, ok := params.Arguments[1].(string)
		if ok {
			if showVerticalFlag == "-show-vertical" {
				showVertical = true
			}
		}
	}

	if err := s.db.Open(); err != nil {
		return nil, err
	}
	defer s.db.Close()

	stmts, err := getStatements(f.Text)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	for _, stmt := range stmts {
		query := strings.TrimSpace(stmt.String())
		if query == "" {
			continue
		}

		if _, isQuery := database.QueryExecType(query, ""); isQuery {
			rows, err := s.db.Query(context.Background(), query)
			if err != nil {
				fmt.Fprintln(buf, err.Error())
				continue
			}
			columns, err := database.Columns(rows)
			if err != nil {
				return nil, err
			}
			stringRows, err := database.ScanRows(rows, len(columns))
			if err != nil {
				return nil, err
			}

			if showVertical {
				table := newVerticalTableWriter(buf)
				table.setHeaders(columns)
				for _, stringRow := range stringRows {
					table.appendRow(stringRow)
				}
				table.render()
			} else {
				table := tablewriter.NewWriter(buf)
				table.SetHeader(columns)
				for _, stringRow := range stringRows {
					table.Append(stringRow)
				}
				table.Render()
			}
			fmt.Fprintf(buf, "%d rows in set", len(stringRows))
			fmt.Fprintln(buf, "")
			fmt.Fprintln(buf, "")

		} else {
			result, err := s.db.Exec(context.Background(), query)
			if err != nil {
				fmt.Fprintln(buf, err.Error())
				continue
			}
			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return nil, err
			}
			fmt.Fprintf(buf, "Query OK, %d row affected", rowsAffected)
			fmt.Fprintln(buf, "")
			fmt.Fprintln(buf, "")
		}
	}
	return buf.String(), nil
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

func getStatements(text string) ([]*ast.Statement, error) {
	src := bytes.NewBuffer([]byte(text))
	p, err := parser.NewParser(src, &dialect.GenericSQLDialect{})
	if err != nil {
		return nil, err
	}
	parsed, err := p.Parse()
	if err != nil {
		return nil, err
	}

	var stmts []*ast.Statement
	for _, node := range parsed.GetTokens() {
		stmt, ok := node.(*ast.Statement)
		if !ok {
			return nil, xerrors.Errorf("invalid type want Statement parsed %T", stmt)
		}
		stmts = append(stmts, stmt)
	}
	return stmts, nil
}

type verticalTableWriter struct {
	writer       io.Writer
	headers      []string
	rows         [][]string
	headerMaxLen int
}

func newVerticalTableWriter(writer io.Writer) *verticalTableWriter {
	return &verticalTableWriter{
		writer: writer,
	}
}

func (vtw *verticalTableWriter) setHeaders(headers []string) {
	vtw.headers = headers
	for _, h := range headers {
		length := len(h)
		if vtw.headerMaxLen < length {
			vtw.headerMaxLen = length
		}
	}
}

func (vtw *verticalTableWriter) appendRow(row []string) {
	vtw.rows = append(vtw.rows, row)
}

func (vtw *verticalTableWriter) render() {
	for rowNum, row := range vtw.rows {
		fmt.Fprintf(vtw.writer, "***************************[ %d. row ]***************************", rowNum+1)
		fmt.Fprintln(vtw.writer, "")
		for colNum, col := range row {
			header := vtw.headers[colNum]

			padHeader := fmt.Sprintf("%"+strconv.Itoa(vtw.headerMaxLen)+"s", header)
			fmt.Fprintf(vtw.writer, "%s | %s", padHeader, col)
			fmt.Fprintln(vtw.writer, "")
		}
	}
}

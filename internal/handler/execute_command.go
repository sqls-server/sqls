package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/sqls-server/sqls/ast"
	"github.com/sqls-server/sqls/internal/database"
	"github.com/sqls-server/sqls/internal/lsp"
	"github.com/sqls-server/sqls/parser"
)

const (
	CommandExecuteQuery     = "executeQuery"
	CommandShowDatabases    = "showDatabases"
	CommandShowSchemas      = "showSchemas"
	CommandShowConnections  = "showConnections"
	CommandSwitchDatabase   = "switchDatabase"
	CommandSwitchConnection = "switchConnections"
	CommandShowTables       = "showTables"
)

func (s *Server) handleTextDocumentCodeAction(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
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
			Title:     "Show Schemas",
			Command:   CommandShowSchemas,
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
		{
			Title:     "Show Tables",
			Command:   CommandShowTables,
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
		return s.executeQuery(ctx, params)
	case CommandShowDatabases:
		return s.showDatabases(ctx, params)
	case CommandShowSchemas:
		return s.showSchemas(ctx, params)
	case CommandShowConnections:
		return s.showConnections(ctx, params)
	case CommandSwitchDatabase:
		return s.switchDatabase(ctx, params)
	case CommandSwitchConnection:
		return s.switchConnections(ctx, params)
	case CommandShowTables:
		return s.showTables(ctx, params)
	}
	return nil, fmt.Errorf("unsupported command: %v", params.Command)
}

func (s *Server) executeQuery(ctx context.Context, params lsp.ExecuteCommandParams) (result interface{}, err error) {
	// parse execute command arguments
	if s.dbConn == nil {
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

	// extract target query
	text := f.Text
	if params.Range != nil {
		text = extractRangeText(
			text,
			params.Range.Start.Line,
			params.Range.Start.Character,
			params.Range.End.Line,
			params.Range.End.Character,
		)
	}
	stmts, err := getStatements(text)
	if err != nil {
		return nil, err
	}

	// execute statements
	buf := new(bytes.Buffer)
	for _, stmt := range stmts {
		query := strings.TrimSpace(stmt.String())
		if query == "" {
			continue
		}

		if _, isQuery := database.QueryExecType(query, ""); isQuery {
			res, err := s.query(ctx, query, showVertical)
			if err != nil {
				return nil, err
			}
			fmt.Fprintln(buf, res)
		} else {
			res, err := s.exec(ctx, query, showVertical)
			if err != nil {
				return nil, err
			}
			fmt.Fprintln(buf, res)
		}
	}
	return buf.String(), nil
}

func extractRangeText(text string, startLine, startChar, endLine, endChar int) string {
	writer := bytes.NewBufferString("")
	scanner := bufio.NewScanner(strings.NewReader(text))

	i := 0
	for scanner.Scan() {
		t := scanner.Text()
		if i >= startLine && i <= endLine {
			st, en := 0, len(t)

			if i == startLine {
				st = startChar
			}
			if i == endLine {
				en = endChar
			}

			writer.Write([]byte(t[st:en]))
			if i != endLine {
				writer.Write([]byte("\n"))
			}
		}
		i++
	}
	return writer.String()
}

func (s *Server) query(ctx context.Context, query string, vertical bool) (string, error) {
	repo, err := s.newDBRepository(ctx)
	if err != nil {
		return "", err
	}
	rows, err := repo.Query(ctx, query)
	if err != nil {
		return "", err
	}
	columns, err := database.Columns(rows)
	if err != nil {
		return "", err
	}
	stringRows, err := database.ScanRows(rows, len(columns))
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	if vertical {
		table := newVerticalTableWriter(buf)
		table.setHeaders(columns)
		for _, stringRow := range stringRows {
			table.appendRow(stringRow)
		}
		table.render()
	} else {
		table := tablewriter.NewWriter(buf)
		// Convert []string to []any for Header
		headers := make([]any, len(columns))
		for i, v := range columns {
			headers[i] = v
		}
		table.Header(headers...)
		for _, stringRow := range stringRows {
			// Convert []string to []any for Append
			row := make([]any, len(stringRow))
			for i, v := range stringRow {
				row[i] = v
			}
			if err := table.Append(row...); err != nil {
				return "", err
			}
		}
		if err := table.Render(); err != nil {
			return "", err
		}
	}
	fmt.Fprintf(buf, "%d rows in set", len(stringRows))
	fmt.Fprintln(buf, "")
	fmt.Fprintln(buf, "")
	return buf.String(), nil
}

func (s *Server) exec(ctx context.Context, query string, vertical bool) (string, error) {
	repo, err := s.newDBRepository(ctx)
	if err != nil {
		return "", err
	}
	result, err := repo.Exec(ctx, query)
	if err != nil {
		return "", err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "Query OK, %d row affected", rowsAffected)
	fmt.Fprintln(buf, "")
	fmt.Fprintln(buf, "")
	return buf.String(), nil
}

func (s *Server) showDatabases(ctx context.Context, params lsp.ExecuteCommandParams) (result interface{}, err error) {
	repo, err := s.newDBRepository(ctx)
	if err != nil {
		return "", err
	}
	databases, err := repo.Databases(ctx)
	if err != nil {
		return nil, err
	}
	return strings.Join(databases, "\n"), nil
}

func (s *Server) showSchemas(ctx context.Context, params lsp.ExecuteCommandParams) (result interface{}, err error) {
	repo, err := s.newDBRepository(ctx)
	if err != nil {
		return "", err
	}
	schemas, err := repo.Schemas(ctx)
	if err != nil {
		return nil, err
	}
	return strings.Join(schemas, "\n"), nil
}

func (s *Server) switchDatabase(ctx context.Context, params lsp.ExecuteCommandParams) (result interface{}, err error) {
	if len(params.Arguments) != 1 {
		return nil, fmt.Errorf("required arguments were not provided: <DB Name>")
	}
	dbName, ok := params.Arguments[0].(string)
	if !ok {
		return nil, fmt.Errorf("specify the db name as a string")
	}

	// Change current database
	s.curDBName = dbName

	// close and reconnection to database
	if err := s.reconnectionDB(ctx); err != nil {
		return nil, err
	}

	return nil, nil
}

func (s *Server) showConnections(ctx context.Context, params lsp.ExecuteCommandParams) (result interface{}, err error) {
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
			case database.ProtoUDP:
				desc = fmt.Sprintf("udp(%s:%d)/%s", conn.Host, conn.Port, conn.DBName)
			case database.ProtoUnix:
				desc = fmt.Sprintf("unix(%s)/%s", conn.Path, conn.DBName)
			case database.ProtoHTTP:
				desc = fmt.Sprintf("http(%s:%d)/%s", conn.Host, conn.Port, conn.DBName)
			}
		}
		res := fmt.Sprintf("%d %s %s %s", i+1, conn.Driver, conn.Alias, desc)
		results = append(results, res)
	}
	return strings.Join(results, "\n"), nil
}

func (s *Server) switchConnections(ctx context.Context, params lsp.ExecuteCommandParams) (result interface{}, err error) {
	if len(params.Arguments) != 1 {
		return nil, fmt.Errorf("required arguments were not provided: <Connection Index>")
	}
	indexStr, ok := params.Arguments[0].(string)
	if !ok {
		return nil, fmt.Errorf("specify the connection index as a number")
	}

	var index int

	cfg := s.getConfig()
	if cfg != nil {
		for i, conn := range cfg.Connections {
			if conn.Alias == indexStr {
				index = i + 1
				break
			}
		}
	}
	if index <= 0 {
		index, _ = strconv.Atoi(indexStr)
	}

	if index <= 0 {
		return nil, fmt.Errorf("specify the connection index as a number, %w", err)
	}
	index = index - 1

	// Reconnect database
	s.curConnectionIndex = index

	// close and reconnection to database
	if err := s.reconnectionDB(ctx); err != nil {
		return nil, err
	}

	return nil, nil
}

func (s *Server) showTables(ctx context.Context, params lsp.ExecuteCommandParams) (result interface{}, err error) {
	repo, err := s.newDBRepository(ctx)
	if err != nil {
		return "", err
	}
	m, err := repo.SchemaTables(ctx)
	if err != nil {
		return nil, err
	}
	schema, err := repo.CurrentSchema(ctx)
	if err != nil {
		return nil, err
	}
	results := []string{}
	for k, vv := range m {
		for _, v := range vv {
			if k != "" {
				if schema != k {
					continue
				}
				results = append(results, k+"."+v)
			} else {
				results = append(results, v)
			}
		}
	}
	return strings.Join(results, "\n"), nil
}

func getStatements(text string) ([]*ast.Statement, error) {
	parsed, err := parser.Parse(text)
	if err != nil {
		return nil, err
	}

	var stmts []*ast.Statement
	for _, node := range parsed.GetTokens() {
		stmt, ok := node.(*ast.Statement)
		if !ok {
			return nil, fmt.Errorf("invalid type want Statement parsed %T", stmt)
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

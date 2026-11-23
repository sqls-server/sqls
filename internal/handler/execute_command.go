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
	"unicode/utf8"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
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

	showVertical, markdown := parseQueryFlags(params.Arguments[1:])

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
			res, err := s.query(ctx, query, showVertical, markdown)
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

func parseQueryFlags(args []interface{}) (showVertical, markdown bool) {
	for _, arg := range args {
		flag, ok := arg.(string)
		if !ok {
			continue
		}
		switch flag {
		case "-show-vertical":
			showVertical = true
		case "-markdown":
			markdown = true
		}
	}
	// the vertical output has no table to convert, so it takes precedence
	if showVertical {
		markdown = false
	}
	return showVertical, markdown
}

func extractRangeText(text string, startLine, startChar, endLine, endChar int) string {
	lines := strings.Split(text, "\n")
	if startLine < 0 {
		startLine = 0
	}
	if endLine >= len(lines) {
		endLine = len(lines) - 1
	}
	if startLine >= len(lines) || startLine > endLine {
		return ""
	}

	var builder strings.Builder
	for i := startLine; i <= endLine; i++ {
		line := strings.TrimSuffix(lines[i], "\r")
		st, en := 0, utf16Len(line)
		if i == startLine {
			st = startChar
		}
		if i == endLine {
			en = endChar
		}
		builder.WriteString(sliceUTF16(line, st, en))
		if i != endLine {
			builder.WriteByte('\n')
		}
	}
	return builder.String()
}

func utf16Len(s string) int {
	length := 0
	for _, r := range s {
		length++
		if r > utf8.MaxRune || r < 0 {
			continue
		}
		if r > 0xFFFF {
			length++
		}
	}
	return length
}

func sliceUTF16(s string, start, end int) string {
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}

	utf16Pos := 0
	startByte, endByte := len(s), len(s)
	startFound, endFound := false, false

	for bytePos, r := range s {
		runeWidth := 1
		if r > 0xFFFF {
			runeWidth = 2
		}

		if !startFound && start <= utf16Pos {
			startByte = bytePos
			startFound = true
		}
		if !endFound && end <= utf16Pos {
			endByte = bytePos
			endFound = true
			break
		}

		nextUTF16Pos := utf16Pos + runeWidth
		if !startFound && start < nextUTF16Pos {
			startByte = bytePos
			startFound = true
		}
		if !endFound && end < nextUTF16Pos {
			endByte = bytePos + utf8.RuneLen(r)
			endFound = true
			break
		}

		utf16Pos = nextUTF16Pos
	}

	if !startFound && start <= utf16Pos {
		startByte = len(s)
	}
	if !endFound {
		endByte = len(s)
	}
	if startByte > endByte {
		startByte = endByte
	}
	return s[startByte:endByte]
}

func (s *Server) query(ctx context.Context, query string, vertical, markdown bool) (string, error) {
	repo, err := s.newDBRepository(ctx)
	if err != nil {
		return "", err
	}
	rows, err := repo.Query(ctx, query)
	if err != nil {
		return "", err
	}
	defer func() { _ = rows.Close() }()
	columns, err := database.Columns(rows)
	if err != nil {
		return "", err
	}
	stringRows, err := database.ScanRows(rows, len(columns))
	if err != nil {
		return "", err
	}

	return renderQueryResult(columns, stringRows, vertical, markdown)
}

func renderQueryResult(columns []string, rows [][]string, vertical, markdown bool) (string, error) {
	buf := new(bytes.Buffer)
	if vertical {
		table := newVerticalTableWriter(buf)
		table.setHeaders(columns)
		for _, row := range rows {
			table.appendRow(row)
		}
		table.render()
	} else {
		if err := renderTable(buf, columns, rows, markdown); err != nil {
			return "", err
		}
		if markdown {
			// a blank line is needed so the footer isn't treated as a table row
			fmt.Fprintln(buf, "")
		}
	}
	fmt.Fprintf(buf, "%d rows in set", len(rows))
	fmt.Fprintln(buf, "")
	fmt.Fprintln(buf, "")
	return buf.String(), nil
}

// markdownEscaper makes sure a value can't add columns or rows, or be interpreted as HTML (which would hide values such as `<nil>`).
var markdownEscaper = strings.NewReplacer(
	`\`, `\\`,
	"|", `\|`,
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	"\r\n", "<br>",
	"\n", "<br>",
	"\r", "<br>",
)

func renderTable(buf *bytes.Buffer, columns []string, rows [][]string, markdown bool) error {
	opts := []tablewriter.Option{
		tablewriter.WithHeaderConfig(tw.CellConfig{
			Formatting: tw.CellFormatting{AutoFormat: tw.Off},
		}),
	}
	if markdown {
		opts = append(opts,
			tablewriter.WithRenderer(renderer.NewMarkdown()),
			tablewriter.WithHeaderAlignment(tw.AlignLeft),
			tablewriter.WithRowAlignment(tw.AlignLeft),
		)
		// a blank line is needed so the table isn't merged with preceding text
		fmt.Fprintln(buf, "")
	}
	table := tablewriter.NewTable(buf, opts...)
	escape := func(v string) string {
		if markdown {
			return markdownEscaper.Replace(v)
		}
		return v
	}
	headers := make([]any, len(columns))
	for i, v := range columns {
		headers[i] = escape(v)
	}
	table.Header(headers...)
	for _, row := range rows {
		vals := make([]any, len(row))
		for i, v := range row {
			vals[i] = escape(v)
		}
		if err := table.Append(vals...); err != nil {
			return err
		}
	}
	return table.Render()
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
		index, err = strconv.Atoi(indexStr)
		if err != nil {
			return nil, fmt.Errorf("specify the connection index as a number, %w", err)
		}
	}

	if index <= 0 {
		return nil, fmt.Errorf("specify the connection index as a number")
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
			return nil, fmt.Errorf("invalid type want Statement parsed %T", node)
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

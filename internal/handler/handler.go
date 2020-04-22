package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/lighttiger2505/sqls/internal/completer"
	"github.com/lighttiger2505/sqls/internal/config"
	"github.com/lighttiger2505/sqls/internal/database"
	"github.com/lighttiger2505/sqls/internal/lsp"
)

type Server struct {
	FileCfg            *config.Config
	WSCfg              *config.Config
	db                 database.Database
	curDBName          string
	curConnectionIndex int
	files              map[string]*File
	completer          *completer.Completer
}

type File struct {
	LanguageID string
	Text       string
}

func NewServer() *Server {
	return &Server{
		files: make(map[string]*File),
	}
}

func (s *Server) init() error {
	s.completer = completer.NewCompleter(s.db)
	if err := s.completer.Init(); err != nil {
		return err
	}
	return nil
}

func panicf(r interface{}, format string, v ...interface{}) error {
	if r != nil {
		// Same as net/http
		const size = 64 << 10
		buf := make([]byte, size)
		buf = buf[:runtime.Stack(buf, false)]
		id := fmt.Sprintf(format, v...)
		log.Printf("panic serving %s: %v\n%s", id, r, string(buf))
		return fmt.Errorf("unexpected panic: %v", r)
	}
	return nil
}

func (s *Server) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	// Prevent any uncaught panics from taking the entire server down.
	defer func() {
		if perr := panicf(recover(), "%v", req.Method); perr != nil {
			err = perr
		}
	}()
	res, err := s.handle(ctx, conn, req)
	if err != nil {
		log.Printf("error serving %+v\n", err)
	}
	return res, err
}
func (s *Server) handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(ctx, conn, req)
	case "initialized":
		return
	case "shutdown":
		return s.handleShutdown(ctx, conn, req)
	case "textDocument/didOpen":
		return s.handleTextDocumentDidOpen(ctx, conn, req)
	case "textDocument/didChange":
		return s.handleTextDocumentDidChange(ctx, conn, req)
	case "textDocument/didSave":
		return s.handleTextDocumentDidSave(ctx, conn, req)
	case "textDocument/didClose":
		return s.handleTextDocumentDidClose(ctx, conn, req)
	case "textDocument/completion":
		return s.handleTextDocumentCompletion(ctx, conn, req)
		// case "textDocument/formatting":
		// 	return h.handleTextDocumentFormatting(ctx, conn, req)
		// case "textDocument/documentSymbol":
		// 	return h.handleTextDocumentSymbol(ctx, conn, req)
	case "textDocument/codeAction":
		return s.handleTextDocumentCodeAction(ctx, conn, req)
	case "workspace/executeCommand":
		return s.handleWorkspaceExecuteCommand(ctx, conn, req)
	case "workspace/didChangeConfiguration":
		return s.handleWorkspaceDidChangeConfiguration(ctx, conn, req)
	}
	return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: fmt.Sprintf("method not supported: %s", req.Method)}
}

func (s *Server) handleInitialize(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.InitializeParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	return lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync:   lsp.TDSKFull,
			HoverProvider:      false,
			CodeActionProvider: true,
			CompletionProvider: &lsp.CompletionOptions{
				TriggerCharacters: []string{"."},
			},
			DefinitionProvider:              false,
			DocumentFormattingProvider:      false,
			DocumentRangeFormattingProvider: false,
		},
	}, nil
}

func (s *Server) handleShutdown(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	if s.db != nil {
		s.db.Close()
	}
	return nil, nil
}

func (s *Server) handleTextDocumentDidOpen(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.DidOpenTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	if err := s.openFile(params.TextDocument.URI, params.TextDocument.LanguageID); err != nil {
		return nil, err
	}
	if err := s.updateFile(params.TextDocument.URI, params.TextDocument.Text); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *Server) handleTextDocumentDidChange(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.DidChangeTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	if err := s.updateFile(params.TextDocument.URI, params.ContentChanges[0].Text); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *Server) handleTextDocumentDidSave(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.DidSaveTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	if params.Text != "" {
		err = s.updateFile(params.TextDocument.URI, params.Text)
	} else {
		err = s.saveFile(params.TextDocument.URI)
	}
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *Server) handleTextDocumentDidClose(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.DidCloseTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	if err := s.closeFile(params.TextDocument.URI); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *Server) openFile(uri string, languageID string) error {
	f := &File{
		Text:       "",
		LanguageID: languageID,
	}
	s.files[uri] = f
	return nil
}

func (s *Server) closeFile(uri string) error {
	delete(s.files, uri)
	return nil
}

func (s *Server) updateFile(uri string, text string) error {
	f, ok := s.files[uri]
	if !ok {
		return fmt.Errorf("document not found: %v", uri)
	}
	f.Text = text
	return nil
}

func (s *Server) saveFile(uri string) error {
	return nil
}

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

func (s *Server) handleWorkspaceDidChangeConfiguration(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	// Update changed configration
	var params lsp.DidChangeConfigurationParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}
	s.WSCfg = params.Settings.SQLS

	// Get the most preferred DB connection settings
	connCfg := s.topConnection()
	if s.curConnectionIndex != 0 {
		connCfg = s.getConnection(s.curConnectionIndex)
	}
	if connCfg == nil {
		return nil, fmt.Errorf("not found database connection config, index %d", s.curConnectionIndex+1)
	}

	// Reconnect DB
	if s.db != nil {
		s.db.Close()
	}
	s.db, err = database.Open(connCfg)
	if err != nil {
		return nil, err
	}
	if s.curDBName != "" {
		if err := s.db.SwitchDB(s.curDBName); err != nil {
			return nil, err
		}
	}

	// Get database, table, columns to complete
	if err := s.init(); err != nil {
		return nil, err
	}
	return nil, nil
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
	if err := s.db.SwitchDB(dbName); err != nil {
		return nil, err
	}
	s.curDBName = dbName
	if err := s.init(); err != nil {
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

	// Get the most preferred DB connection settings
	connCfg := s.getConnection(index)
	if connCfg == nil {
		return nil, fmt.Errorf("not found database connection config, index %d", index+1)
	}
	s.curConnectionIndex = index

	// Reconnect DB
	if s.db != nil {
		s.db.Close()
	}
	s.db, err = database.Open(connCfg)
	if err != nil {
		return nil, err
	}
	if s.curDBName != "" {
		if err := s.db.SwitchDB(s.curDBName); err != nil {
			return nil, err
		}
	}

	// Get database infomations(databases, tables, columns) to complete
	if err := s.init(); err != nil {
		return nil, err
	}
	return nil, nil
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

func (s *Server) topConnection() *database.Config {
	cfg := s.getConfig()
	if len(cfg.Connections) == 0 {
		return nil
	}
	return cfg.Connections[0]
}

func (s *Server) getConnection(index int) *database.Config {
	cfg := s.getConfig()
	if index < 0 && len(cfg.Connections) <= index {
		return nil
	}
	return cfg.Connections[index]
}

func (s *Server) getConfig() *config.Config {
	var cfg *config.Config
	if s.FileCfg != nil {
		cfg = s.FileCfg
	} else {
		cfg = s.WSCfg
	}
	return cfg
}

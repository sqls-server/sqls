package handler

import (
	"context"
	"log"
	"net"
	"reflect"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/lighttiger2505/sqls/internal/config"
	"github.com/lighttiger2505/sqls/internal/lsp"
)

const testFileURI = "file:///Users/octref/Code/css-test/test.sql"

type TestContext struct {
	h          jsonrpc2.Handler
	conn       *jsonrpc2.Conn
	connServer *jsonrpc2.Conn
	server     *Server
	ctx        context.Context
}

func newTestContext() *TestContext {
	server := NewServer()
	handler := jsonrpc2.HandlerWithError(server.Handle)
	ctx := context.Background()
	return &TestContext{
		h:      handler,
		ctx:    ctx,
		server: server,
	}
}

func (tx *TestContext) setup(t *testing.T) {
	t.Helper()
	tx.initServer(t)
}

func (tx *TestContext) tearDown() {
	if tx.conn != nil {
		if err := tx.conn.Close(); err != nil {
			log.Fatal("conn.Close:", err)
		}
	}

	if tx.connServer != nil {
		if err := tx.connServer.Close(); err != nil {
			log.Fatal("connServer.Close:", err)
		}
	}
}

func (tx *TestContext) initServer(t *testing.T) {
	t.Helper()

	// Prepare the server and client connection.
	client, server := net.Pipe()
	tx.connServer = jsonrpc2.NewConn(tx.ctx, jsonrpc2.NewBufferedStream(server, jsonrpc2.VSCodeObjectCodec{}), tx.h)
	tx.conn = jsonrpc2.NewConn(tx.ctx, jsonrpc2.NewBufferedStream(client, jsonrpc2.VSCodeObjectCodec{}), tx.h)

	// Initialize Langage Server
	params := lsp.InitializeParams{
		InitializationOptions: lsp.InitializeOptions{},
	}
	if err := tx.conn.Call(tx.ctx, "initialize", params, nil); err != nil {
		t.Fatal("conn.Call initialize:", err)
	}
}

func (tx *TestContext) addWorkspaceConfig(t *testing.T, cfg *config.Config) {
	didChangeConfigurationParams := lsp.DidChangeConfigurationParams{
		Settings: struct {
			SQLS *config.Config "json:\"sqls\""
		}{
			SQLS: cfg,
		},
	}
	if err := tx.conn.Call(tx.ctx, "workspace/didChangeConfiguration", didChangeConfigurationParams, nil); err != nil {
		t.Fatal("conn.Call workspace/didChangeConfiguration:", err)
	}
}

func (tx *TestContext) textDocumentDidOpen(t *testing.T, uri, input string) {
	didOpenParams := lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:        testFileURI,
			LanguageID: "sql",
			Version:    0,
			Text:       input,
		},
	}
	if err := tx.conn.Call(tx.ctx, "textDocument/didOpen", didOpenParams, nil); err != nil {
		t.Fatal("conn.Call textDocument/didOpen:", err)
	}
	tx.testFile(t, didOpenParams.TextDocument.URI, didOpenParams.TextDocument.Text)
}

func TestInitialized(t *testing.T) {
	tx := newTestContext()
	tx.setup(t)
	defer tx.tearDown()

	want := lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: lsp.TDSKFull,
			HoverProvider:    true,
			CompletionProvider: &lsp.CompletionOptions{
				TriggerCharacters: []string{"(", "."},
			},
			SignatureHelpProvider: &lsp.SignatureHelpOptions{
				TriggerCharacters:   []string{"(", ","},
				RetriggerCharacters: []string{"(", ","},
				WorkDoneProgressOptions: lsp.WorkDoneProgressOptions{
					WorkDoneProgress: false,
				},
			},
			CodeActionProvider:              true,
			DefinitionProvider:              false,
			DocumentFormattingProvider:      true,
			DocumentRangeFormattingProvider: true,
			RenameProvider:                  true,
		},
	}
	var got lsp.InitializeResult
	params := lsp.InitializeParams{
		InitializationOptions: lsp.InitializeOptions{},
	}
	if err := tx.conn.Call(tx.ctx, "initialize", params, &got); err != nil {
		t.Fatal("conn.Call initialize:", err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("not match \n%+v\n%+v", want, got)
	}
}

func TestFileWatch(t *testing.T) {
	tx := newTestContext()
	tx.setup(t)
	defer tx.tearDown()

	uri := "file:///Users/octref/Code/css-test/test.sql"
	openText := "SELECT * FROM todo ORDER BY id ASC"
	changeText := "SELECT * FROM todo ORDER BY name ASC"

	didOpenParams := lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:        uri,
			LanguageID: "sql",
			Version:    0,
			Text:       openText,
		},
	}
	if err := tx.conn.Call(tx.ctx, "textDocument/didOpen", didOpenParams, nil); err != nil {
		t.Fatal("conn.Call textDocument/didOpen:", err)
	}
	tx.testFile(t, didOpenParams.TextDocument.URI, didOpenParams.TextDocument.Text)

	didChangeParams := lsp.DidChangeTextDocumentParams{
		TextDocument: lsp.VersionedTextDocumentIdentifier{
			URI:     uri,
			Version: 1,
		},
		ContentChanges: []lsp.TextDocumentContentChangeEvent{
			lsp.TextDocumentContentChangeEvent{
				Range: lsp.Range{
					Start: lsp.Position{
						Line:      1,
						Character: 1,
					},
					End: lsp.Position{
						Line:      1,
						Character: 1,
					},
				},
				RangeLength: 1,
				Text:        changeText,
			},
		},
	}
	if err := tx.conn.Call(tx.ctx, "textDocument/didChange", didChangeParams, nil); err != nil {
		t.Fatal("conn.Call textDocument/didChange:", err)
	}
	tx.testFile(t, didChangeParams.TextDocument.URI, didChangeParams.ContentChanges[0].Text)

	didSaveParams := lsp.DidSaveTextDocumentParams{
		Text:         openText,
		TextDocument: lsp.TextDocumentIdentifier{URI: uri},
	}
	if err := tx.conn.Call(tx.ctx, "textDocument/didSave", didSaveParams, nil); err != nil {
		t.Fatal("conn.Call textDocument/didSave:", err)
	}
	tx.testFile(t, didSaveParams.TextDocument.URI, didSaveParams.Text)

	didCloseParams := lsp.DidCloseTextDocumentParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: uri},
	}
	if err := tx.conn.Call(tx.ctx, "textDocument/didClose", didCloseParams, nil); err != nil {
		t.Fatal("conn.Call textDocument/didClose:", err)
	}
	_, ok := tx.server.files[didCloseParams.TextDocument.URI]
	if ok {
		t.Errorf("found opened file. URI:%s", didCloseParams.TextDocument.URI)
	}
}

func (tx *TestContext) testFile(t *testing.T, uri, text string) {
	f, ok := tx.server.files[uri]
	if !ok {
		t.Errorf("not found opened file. URI:%s", uri)
	}
	if f.Text != text {
		t.Errorf("not match %s. got: %s", text, f.Text)
	}
}

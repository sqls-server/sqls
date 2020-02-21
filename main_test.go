package main

import (
	"context"
	"log"
	"net"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/jsonrpc2"
)

type TestContext struct {
	h          jsonrpc2.Handler
	conn       *jsonrpc2.Conn
	connServer *jsonrpc2.Conn
	server     *Server
	ctx        context.Context
}

func newTestContext() *TestContext {
	server := NewServer()
	handler := jsonrpc2.HandlerWithError(server.handle)
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
	params := InitializeParams{
		InitializationOptions: InitializeOptions{},
	}
	if err := tx.conn.Call(tx.ctx, "initialize", params, nil); err != nil {
		t.Fatal("conn.Call initialize:", err)
	}
}

func TestInitialized(t *testing.T) {
	tx := newTestContext()
	tx.setup(t)
	defer tx.tearDown()

	want := InitializeResult{
		ServerCapabilities{
			TextDocumentSync: TDSKFull,
			HoverProvider:    false,
			CompletionProvider: &CompletionOptions{
				TriggerCharacters: []string{"."},
			},
			DefinitionProvider:              false,
			DocumentFormattingProvider:      false,
			DocumentRangeFormattingProvider: false,
		},
	}
	var got InitializeResult
	params := InitializeParams{
		InitializationOptions: InitializeOptions{},
	}
	if err := tx.conn.Call(tx.ctx, "initialize", params, &got); err != nil {
		t.Fatal("conn.Call initialize:", err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("not match \n%v\n%v", want, got)
	}
}

func TestFileWatch(t *testing.T) {
	tx := newTestContext()
	tx.setup(t)
	defer tx.tearDown()

	uri := "file:///Users/octref/Code/css-test/test.sql"
	openText := "SELECT * FROM todo ORDER BY id ASC"
	changeText := "SELECT * FROM todo ORDER BY name ASC"

	didOpenParams := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
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

	didChangeParams := DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			URI:     uri,
			Version: 1,
		},
		ContentChanges: []TextDocumentContentChangeEvent{
			TextDocumentContentChangeEvent{
				Range: Range{
					Start: Position{
						Line:      1,
						Character: 1,
					},
					End: Position{
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

	didSaveParams := DidSaveTextDocumentParams{
		Text:         openText,
		TextDocument: TextDocumentIdentifier{uri},
	}
	if err := tx.conn.Call(tx.ctx, "textDocument/didSave", didSaveParams, nil); err != nil {
		t.Fatal("conn.Call textDocument/didSave:", err)
	}
	tx.testFile(t, didSaveParams.TextDocument.URI, didSaveParams.Text)

	didCloseParams := DidCloseTextDocumentParams{TextDocumentIdentifier{uri}}
	if err := tx.conn.Call(tx.ctx, "textDocument/didClose", didCloseParams, nil); err != nil {
		t.Fatal("conn.Call textDocument/didClose:", err)
	}
	_, ok := tx.server.files[didCloseParams.TextDocument.URI]
	if ok {
		t.Errorf("found opened file. URI:%s", didCloseParams.TextDocument.URI)
	}
}

func TestComplete(t *testing.T) {
	tx := newTestContext()
	tx.setup(t)
	defer tx.tearDown()

	didChangeConfigurationParams := DidChangeConfigurationParams{
		Settings: struct {
			SQLS struct {
				Driver         string "json:\"driver\""
				DataSourceName string "json:\"dataSourceName\""
			} "json:\"sqls\""
		}{
			SQLS: struct {
				Driver         string "json:\"driver\""
				DataSourceName string "json:\"dataSourceName\""
			}{
				Driver:         "mock",
				DataSourceName: "",
			},
		},
	}
	if err := tx.conn.Call(tx.ctx, "workspace/didChangeConfiguration", didChangeConfigurationParams, nil); err != nil {
		t.Fatal("conn.Call workspace/didChangeConfiguration:", err)
	}

	uri := "file:///Users/octref/Code/css-test/test.sql"
	testcases := []struct {
		name  string
		input string
		line  int
		col   int
		want  []CompletionItem
	}{
		{
			name:  "select identifier",
			input: "select  from city",
			line:  0,
			col:   7,
			want: []CompletionItem{
				{
					Label:  "ID",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "Name",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "CountryCode",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "District",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "Population",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "city",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
				{
					Label:  "country",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
				{
					Label:  "countrylanguage",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
			},
		},
		{
			name:  "select identifier with table alias",
			input: "select  from city as c",
			line:  0,
			col:   7,
			want: []CompletionItem{
				{
					Label:  "ID",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "Name",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "CountryCode",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "District",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "Population",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "c",
					Kind:   FieldCompletion,
					Detail: AliasDetailTemplate,
				},
				{
					Label:  "city",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
				{
					Label:  "country",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
				{
					Label:  "countrylanguage",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
			},
		},
		{
			name:  "select identifier filterd",
			input: "select Cou from city",
			line:  0,
			col:   10,
			want: []CompletionItem{
				{
					Label:  "CountryCode",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "country",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
				{
					Label:  "countrylanguage",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
			},
		},
		{
			name:  "select has parent table identifier",
			input: "select city. from city",
			line:  0,
			col:   12,
			want: []CompletionItem{
				{
					Label:  "ID",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "Name",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "CountryCode",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "District",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "Population",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
			},
		},
		{
			name:  "select identifier list",
			input: "select id, cou from city",
			line:  0,
			col:   14,
			want: []CompletionItem{
				{
					Label:  "CountryCode",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "country",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
				{
					Label:  "countrylanguage",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
			},
		},
		{
			name:  "select has aliased table identifier",
			input: "select c. from city as c",
			line:  0,
			col:   9,
			want: []CompletionItem{
				{
					Label:  "ID",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "Name",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "CountryCode",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "District",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "Population",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
			},
		},
		{
			name:  "from identifier",
			input: "select CountryCode from ",
			line:  0,
			col:   24,
			want: []CompletionItem{
				{
					Label:  "city",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
				{
					Label:  "country",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
				{
					Label:  "countrylanguage",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
			},
		},
		{
			name:  "from identifier filterd",
			input: "select CountryCode from co",
			line:  0,
			col:   26,
			want: []CompletionItem{
				{
					Label:  "country",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
				{
					Label:  "countrylanguage",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
			},
		},
		{
			name:  "join identifier",
			input: "select CountryCode from city left join ",
			line:  0,
			col:   39,
			want: []CompletionItem{
				{
					Label:  "ID",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "Name",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "CountryCode",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "District",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "Population",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "city",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
				{
					Label:  "country",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
				{
					Label:  "countrylanguage",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
			},
		},
		{
			name:  "join identifier filterd",
			input: "select CountryCode from city left join co",
			line:  0,
			col:   41,
			want: []CompletionItem{
				{
					Label:  "CountryCode",
					Kind:   FieldCompletion,
					Detail: ColumnDetailTemplate,
				},
				{
					Label:  "country",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
				{
					Label:  "countrylanguage",
					Kind:   FieldCompletion,
					Detail: TableDetailTemplate,
				},
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			// Open dummy file
			didOpenParams := DidOpenTextDocumentParams{
				TextDocument: TextDocumentItem{
					URI:        uri,
					LanguageID: "sql",
					Version:    0,
					Text:       tt.input,
				},
			}
			if err := tx.conn.Call(tx.ctx, "textDocument/didOpen", didOpenParams, nil); err != nil {
				t.Fatal("conn.Call textDocument/didOpen:", err)
			}
			tx.testFile(t, didOpenParams.TextDocument.URI, didOpenParams.TextDocument.Text)
			// Create completion params
			commpletionParams := CompletionParams{
				TextDocumentPositionParams: TextDocumentPositionParams{
					TextDocument: TextDocumentIdentifier{
						URI: uri,
					},
					Position: Position{
						Line:      tt.line,
						Character: tt.col,
					},
				},
				CompletionContext: CompletionContext{
					TriggerKind:      0,
					TriggerCharacter: nil,
				},
			}

			var got []CompletionItem
			if err := tx.conn.Call(tx.ctx, "textDocument/completion", commpletionParams, &got); err != nil {
				t.Fatal("conn.Call textDocument/completion:", err)
			}

			if d := cmp.Diff(tt.want, got); d != "" {
				t.Errorf("completion item diff: %s", d)
			}
		})
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

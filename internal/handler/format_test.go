package handler

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lighttiger2505/sqls/internal/lsp"
)

func TestFormatting(t *testing.T) {
	tx := newTestContext()
	tx.initServer(t)
	defer tx.tearDown()

	uri := "file:///Users/octref/Code/css-test/test.sql"

	type formattingTestCase struct {
		name  string
		input string
		want  string
	}

	testDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	testFileInfos, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}

	testCase := []formattingTestCase{}

	// Add golden file test
	const (
		inputFileSuffix  = ".input.sql"
		goldenFileSuffix = ".golden.sql"
	)
	for _, testFileInfo := range testFileInfos {
		inputFileName := testFileInfo.Name()
		if !strings.HasSuffix(inputFileName, inputFileSuffix) {
			continue
		}

		testName := testFileInfo.Name()[:len(inputFileName)-len(inputFileSuffix)]
		inputPath := filepath.Join(testDir, "testdata", inputFileName)
		goldenPath := filepath.Join(testDir, "testdata", testName+goldenFileSuffix)

		input, err := ioutil.ReadFile(inputPath)
		if err != nil {
			t.Errorf("Cannot read input file, Path=%s, Err=%+v", inputPath, err)
			continue
		}
		golden, err := ioutil.ReadFile(goldenPath)
		if err != nil {
			t.Errorf("Cannot read input file, Path=%s, Err=%+v", goldenPath, err)
			continue
		}
		testCase = append(testCase, formattingTestCase{
			name:  testName,
			input: string(input),
			want:  string(golden)[:len(string(golden))-len("\n")],
		})
	}

	// Add minimal case test
	minimalTestCase := []formattingTestCase{
		{
			name:  "multi keyword",
			input: "  inner   join  ",
			want:  "inner join",
		},
		{
			name:  "aliased",
			input: "foo   as   f",
			want:  "foo as f",
		},
		{
			name:  "member identifer",
			input: "foo.id",
			want:  "foo.id",
		},
		{
			name:  "operator",
			input: "1+ 2  -   3    *     4",
			want:  "1 + 2 - 3 * 4",
		},
		{
			name:  "comparison",
			input: "1 <  2",
			want:  "1 < 2",
		},
		// {
		// 	name:  "parenthesis",
		// 	input: "( 1  +   2    )     =      3",
		// 	want:  "(1 + 2) = 3",
		// },
		{
			name:  "identifier list",
			input: "1 ,  2   ,    3     ,      4",
			want:  "1,\n2,\n3,\n4",
		},
	}
	testCase = append(testCase, minimalTestCase...)

	for _, tt := range testCase {
		t.Run(tt.name, func(t *testing.T) {
			// Open dummy file
			didOpenParams := lsp.DidOpenTextDocumentParams{
				TextDocument: lsp.TextDocumentItem{
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
			formattingParams := lsp.DocumentFormattingParams{
				TextDocument: lsp.TextDocumentIdentifier{
					URI: uri,
				},
				Options: lsp.FormattingOptions{
					TabSize:                0.0,
					InsertSpaces:           false,
					TrimTrailingWhitespace: false,
					InsertFinalNewline:     false,
					TrimFinalNewlines:      false,
				},
				WorkDoneProgressParams: lsp.WorkDoneProgressParams{
					WorkDoneToken: nil,
				},
			}

			var got []lsp.TextEdit
			if err := tx.conn.Call(tx.ctx, "textDocument/formatting", formattingParams, &got); err != nil {
				t.Fatal("conn.Call textDocument/formatting:", err)
			}
			if diff := cmp.Diff(tt.want, got[0].NewText); diff != "" {
				t.Errorf("unmatch (- want, + got):\n%s", diff)
				t.Errorf("unmatch\nwant: %q\ngot : %q", tt.want, got[0].NewText)
			}
		})
	}
}

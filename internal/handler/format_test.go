package handler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sqls-server/sqls/internal/config"
	"github.com/sqls-server/sqls/internal/lsp"
)

var formattingOptionTab = lsp.FormattingOptions{
	TabSize:                0.0,
	InsertSpaces:           false,
	TrimTrailingWhitespace: false,
	InsertFinalNewline:     false,
	TrimFinalNewlines:      false,
}

var formattingOptionIndentSpace2 = lsp.FormattingOptions{
	TabSize:                2.0,
	InsertSpaces:           true,
	TrimTrailingWhitespace: false,
	InsertFinalNewline:     false,
	TrimFinalNewlines:      false,
}

var formattingOptionIndentSpace4 = lsp.FormattingOptions{
	TabSize:                4.0,
	InsertSpaces:           true,
	TrimTrailingWhitespace: false,
	InsertFinalNewline:     false,
	TrimFinalNewlines:      false,
}

var upperCaseConfig = &config.Config{
	LowercaseKeywords: false,
}

var lowerCaseConfig = &config.Config{
	LowercaseKeywords: true,
}

type formattingTestCase struct {
	name  string
	input string
	want  string
}

func testFormatting(t *testing.T, testCases []formattingTestCase, options lsp.FormattingOptions, cfg *config.Config) {
	tx := newTestContext()
	tx.initServer(t)
	defer tx.tearDown()

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
	uri := "file:///Users/octref/Code/css-test/test.sql"
	for _, tt := range testCases {
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
				Options: options,
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

func TestFormattingBase(t *testing.T) {
	testCase, err := loadFormatTestCaseByTestdata("format")
	if err != nil {
		t.Fatal(err)
	}
	testFormatting(t, testCase, formattingOptionTab, lowerCaseConfig)
}

func TestFormattingMinimal(t *testing.T) {
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
			name:  "member identifier",
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
	testFormatting(t, minimalTestCase, formattingOptionTab, lowerCaseConfig)
}

func TestFormattingWithOptionSpace2(t *testing.T) {
	testCase, err := loadFormatTestCaseByTestdata("format_option_space2")
	if err != nil {
		t.Fatal(err)
	}
	testFormatting(t, testCase, formattingOptionIndentSpace2, lowerCaseConfig)
}

func TestFormattingWithOptionSpace4(t *testing.T) {
	testCase, err := loadFormatTestCaseByTestdata("format_option_space4")
	if err != nil {
		t.Fatal(err)
	}
	testFormatting(t, testCase, formattingOptionIndentSpace4, lowerCaseConfig)
}

func TestFormattingWithOptionUpper(t *testing.T) {
	testCase, err := loadFormatTestCaseByTestdata("upper_case")
	if err != nil {
		t.Fatal(err)
	}
	testFormatting(t, testCase, formattingOptionTab, upperCaseConfig)
}

func loadFormatTestCaseByTestdata(targetDir string) ([]formattingTestCase, error) {
	packageDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	testDir := filepath.Join(packageDir, "testdata", targetDir)
	testFileInfos, err := os.ReadDir(testDir)
	if err != nil {
		return nil, err
	}

	testCase := []formattingTestCase{}
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
		inputPath := filepath.Join(testDir, inputFileName)
		goldenPath := filepath.Join(testDir, testName+goldenFileSuffix)

		input, err := os.ReadFile(inputPath)
		if err != nil {
			return nil, fmt.Errorf("Cannot read input file, Path=%s, Err=%+v", inputPath, err)
		}
		golden, err := os.ReadFile(goldenPath)
		if err != nil {
			return nil, fmt.Errorf("Cannot read input file, Path=%s, Err=%+v", goldenPath, err)
		}
		testCase = append(testCase, formattingTestCase{
			name:  testName,
			input: string(input),
			want:  string(golden),
		})
	}
	return testCase, nil
}

package main

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSimple(t *testing.T) {
	input := "SELECT * FROM hogehoge WHERE a = 'abc'"
	parser := &Parser{}
	got, err := parser.parse(input)
	if err != nil {
		t.Fatalf("error, %s", err.Error())
	}

	want := "select"
	if got != want {
		t.Fatalf("want %v, but %v:", want, got)
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		line  int
		char  int
		want  []CompletionType
	}{
		{
			name:  "SELECT | FROM `city`",
			input: "SELECT  FROM `city`",
			line:  1,
			char:  8,
			want: []CompletionType{
				CompletionTypeColumn,
				CompletionTypeTable,
				CompletionTypeView,
				CompletionTypeFunction,
			},
		},
		// {
		// 	name:  "SELECT `city`.| FROM `city`",
		// 	input: "SELECT `city`.  FROM `city`",
		// 	line:  1,
		// 	char:  15,
		// 	want: []CompletionType{
		// 		CompletionTypeColumn,
		// 		CompletionTypeTable,
		// 		CompletionTypeView,
		// 		CompletionTypeFunction,
		// 	},
		// },
		{
			name:  "SELECT * FROM | ",
			input: "SELECT * FROM   ",
			line:  1,
			char:  15,
			want: []CompletionType{
				CompletionTypeColumn,
				CompletionTypeTable,
				CompletionTypeView,
				CompletionTypeFunction,
			},
		},
		{
			name:  "SELECT * FROM `city` WHERE | ",
			input: "SELECT * FROM `city` WHERE   ",
			line:  1,
			char:  28,
			want: []CompletionType{
				CompletionTypeColumn,
				CompletionTypeTable,
				CompletionTypeView,
				CompletionTypeFunction,
			},
		},
		{
			name:  "SELECT col as | FROM `city`",
			input: "SELECT col as  FROM `city`",
			line:  1,
			char:  15,
			want:  []CompletionType{},
		},
		{
			name:  "SELECT * FROM `city` as | ",
			input: "SELECT * FROM `city` as  ",
			line:  1,
			char:  25,
			want:  []CompletionType{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &Parser{}
			got, err := parser.parse(tt.input, tt.line, tt.char)
			if err != nil {
				t.Fatalf("error, %s", err.Error())
			}

			want := tt.want
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("want %v, but %v:", want, got)
			}
		})
	}
}

func TestGetBeforeCursorText(t *testing.T) {
	input := `SELECT
a, b, c
FROM
hogetable
`
	tests := []struct {
		in   string
		line int
		char int
		out  string
	}{
		{input, 1, 2, "SE"},
		{input, 2, 3, "SELECT\na, "},
		{input, 3, 4, "SELECT\na, b, c\nFROM"},
		{input, 4, 5, "SELECT\na, b, c\nFROM\nhoget"},
	}
	for _, tt := range tests {
		got := getBeforeCursorText(tt.in, tt.line, tt.char)
		if tt.out != got {
			t.Errorf("want %#v, got %#v", tt.out, got)
		}
	}
}

func TestGetLastWord(t *testing.T) {
	input := `SELECT
    a, b, c
FROM
    hogetable
`
	tests := []struct {
		in   string
		line int
		char int
		out  string
	}{
		{input, 1, 2, "SE"},
		{input, 2, 3, ""},
		{input, 3, 4, "FROM"},
		{input, 4, 5, "h"},
	}
	for _, tt := range tests {
		got := getLastWord(tt.in, tt.line, tt.char)
		if tt.out != got {
			t.Errorf("want %#v, got %#v", tt.out, got)
		}
	}
}

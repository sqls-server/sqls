package main

import (
	"reflect"
	"testing"
)

func Test_Parse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		line  int
		char  int
		want  []CompletionType
	}{
		{
			name:  "SELECT | FROM `city`",
			input: "SELECT  FROM def",
			line:  1,
			char:  7,
			want: []CompletionType{
				CompletionTypeColumn,
				CompletionTypeTable,
				CompletionTypeView,
				CompletionTypeFunction,
			},
		},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parse(tt.input, tt.line, tt.char)
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

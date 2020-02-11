package main

import (
	"testing"
)

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
		name string
		in   string
		line int
		char int
		out  string
	}{
		{"", "SELECT  FROM def", 1, 7, ""},
		{"", input, 1, 2, "SE"},
		{"", input, 2, 3, ""},
		{"", input, 3, 4, "FROM"},
		{"", input, 3, 6, ""},
		{"", input, 4, 5, "h"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getLastWord(tt.in, tt.line, tt.char)
			if tt.out != got {
				t.Errorf("want %#v, got %#v", tt.out, got)
			}
		})
	}
}

func Test_completionTypeIs(t *testing.T) {
	type args struct {
	}
	tests := []struct {
		name            string
		completionTypes []CompletionType
		expect          CompletionType
		want            bool
	}{
		{
			completionTypes: []CompletionType{
				CompletionTypeColumn,
			},
			expect: CompletionTypeColumn,
			want:   true,
		},
		{
			completionTypes: []CompletionType{
				CompletionTypeTable,
				CompletionTypeView,
				CompletionTypeFunction,
				CompletionTypeColumn,
			},
			expect: CompletionTypeColumn,
			want:   true,
		},
		{
			completionTypes: []CompletionType{
				CompletionTypeTable,
				CompletionTypeView,
				CompletionTypeFunction,
			},
			expect: CompletionTypeColumn,
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := completionTypeIs(tt.completionTypes, tt.expect); got != tt.want {
				t.Errorf("completionTypeIs() = %v, want %v", got, tt.want)
			}
		})
	}
}

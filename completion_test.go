package main

import (
	"testing"
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

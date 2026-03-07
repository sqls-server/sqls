package database

import "testing"

func Test_sqlValToString_nilTypedPointer(t *testing.T) {
	// Regression test: a typed nil pointer (e.g. (*string)(nil)) stored in
	// interface{} should return empty string, not panic.
	var s *string
	var iface interface{} = s
	pointer := &iface

	got, err := sqlValToString(pointer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string for nil *string, got %q", got)
	}
}

func Test_sqlValToString_nilInterface(t *testing.T) {
	got, err := sqlValToString(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string for nil, got %q", got)
	}
}

func Test_sqlValToString_validPointer(t *testing.T) {
	s := "hello"
	var iface interface{} = &s
	pointer := &iface

	got, err := sqlValToString(pointer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Errorf("expected %q, got %q", "hello", got)
	}
}

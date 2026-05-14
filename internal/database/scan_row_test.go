package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sync"
	"testing"
)

func TestScanRows_returnsRowsErr(t *testing.T) {
	db := openScanRowsTestDB(t)
	defer db.Close()

	rows, err := db.QueryContext(context.Background(), "SELECT 1")
	if err != nil {
		t.Fatalf("QueryContext() error = %v", err)
	}
	defer rows.Close()

	columns, err := Columns(rows)
	if err != nil {
		t.Fatalf("Columns() error = %v", err)
	}

	_, err = ScanRows(rows, len(columns))
	if !errors.Is(err, errScanRowsTest) {
		t.Fatalf("ScanRows() error = %v, want %v", err, errScanRowsTest)
	}
}

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

var (
	registerScanRowsTestDriverOnce sync.Once
	errScanRowsTest                = errors.New("scan rows test error")
)

func openScanRowsTestDB(t *testing.T) *sql.DB {
	t.Helper()

	registerScanRowsTestDriverOnce.Do(func() {
		sql.Register("scan_rows_test", scanRowsTestDriver{})
	})

	db, err := sql.Open("scan_rows_test", "")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	return db
}

type scanRowsTestDriver struct{}

func (scanRowsTestDriver) Open(string) (driver.Conn, error) {
	return scanRowsTestConn{}, nil
}

type scanRowsTestConn struct{}

func (scanRowsTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (scanRowsTestConn) Close() error {
	return nil
}

func (scanRowsTestConn) Begin() (driver.Tx, error) {
	return nil, errors.New("not implemented")
}

func (scanRowsTestConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return &scanRowsTestRows{}, nil
}

type scanRowsTestRows struct {
	calls int
}

func (r *scanRowsTestRows) Columns() []string {
	return []string{"value"}
}

func (r *scanRowsTestRows) Close() error {
	return nil
}

func (r *scanRowsTestRows) Next(dest []driver.Value) error {
	switch r.calls {
	case 0:
		dest[0] = "ok"
		r.calls++
		return nil
	case 1:
		r.calls++
		return errScanRowsTest
	default:
		return io.EOF
	}
}

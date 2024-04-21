package handler

import (
	"testing"

	"github.com/sqls-server/sqls/internal/lsp"
)

func TestCleanRustInjections(t *testing.T) {
	file := &File{
		LanguageID: `rust`,
		Text:       `fn main() { sqlx::query("select * from users").fetch_all(&mut conn).await; }`,
	}
	result := cleanRustInjections(file, lsp.Position{Line: 0, Character: 0})

	expected := `                         select * from users                                `

	if result != expected {
		t.Errorf("got %s, want %s", result, expected)
	}
}

package lsp

import (
	"context"
	"log"

	"github.com/sourcegraph/jsonrpc2"
)

type Messenger interface {
	ShowLog(context.Context, string) error
	ShowInfo(context.Context, string) error
	ShowWarning(context.Context, string) error
	ShowError(context.Context, string) error
}

type LspMessenger struct {
	conn *jsonrpc2.Conn
}

func NewLspMessenger(conn *jsonrpc2.Conn) Messenger {
	return &LspMessenger{
		conn: conn,
	}
}

func (m *LspMessenger) ShowLog(ctx context.Context, message string) error {
	log.Println("Send Message:", message)
	params := &ShowMessageParams{
		Type:    Log,
		Message: message,
	}
	return m.conn.Notify(ctx, "window/showMessage", params)
}

func (m *LspMessenger) ShowInfo(ctx context.Context, message string) error {
	log.Println("Send Message:", message)
	params := &ShowMessageParams{
		Type:    Info,
		Message: message,
	}
	return m.conn.Notify(ctx, "window/showMessage", params)
}

func (m *LspMessenger) ShowWarning(ctx context.Context, message string) error {
	log.Println("Send Message:", message)
	params := &ShowMessageParams{
		Type:    Warning,
		Message: message,
	}
	return m.conn.Notify(ctx, "window/showMessage", params)
}

func (m *LspMessenger) ShowError(ctx context.Context, message string) error {
	log.Println("Send Message:", message)
	params := &ShowMessageParams{
		Type:    Error,
		Message: message,
	}
	return m.conn.Notify(ctx, "window/showMessage", params)
}

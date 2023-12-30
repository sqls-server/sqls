package lsp

import (
	"context"
	"log"

	"github.com/sourcegraph/jsonrpc2"
)

type MessageDisplayer interface {
	ShowLog(context.Context, string) error
	ShowInfo(context.Context, string) error
	ShowWarning(context.Context, string) error
	ShowError(context.Context, string) error
}

type Messenger struct {
	conn *jsonrpc2.Conn
}

func NewMessenger(conn *jsonrpc2.Conn) MessageDisplayer {
	return &Messenger{
		conn: conn,
	}
}

func (m *Messenger) ShowLog(ctx context.Context, message string) error {
	log.Println("Send Message:", message)
	params := &ShowMessageParams{
		Type:    Log,
		Message: message,
	}
	return m.conn.Notify(ctx, "window/showMessage", params)
}

func (m *Messenger) ShowInfo(ctx context.Context, message string) error {
	log.Println("Send Message:", message)
	params := &ShowMessageParams{
		Type:    Info,
		Message: message,
	}
	return m.conn.Notify(ctx, "window/showMessage", params)
}

func (m *Messenger) ShowWarning(ctx context.Context, message string) error {
	log.Println("Send Message:", message)
	params := &ShowMessageParams{
		Type:    Warning,
		Message: message,
	}
	return m.conn.Notify(ctx, "window/showMessage", params)
}

func (m *Messenger) ShowError(ctx context.Context, message string) error {
	log.Println("Send Message:", message)
	params := &ShowMessageParams{
		Type:    Error,
		Message: message,
	}
	return m.conn.Notify(ctx, "window/showMessage", params)
}

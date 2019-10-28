package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sourcegraph/jsonrpc2"
)

var (
	logfile string
)

func main() {
	flag.StringVar(&logfile, "log", "", "logfile")
	flag.Parse()

	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(1)
	}
	log.Println("sqls: reading on stdin, writing on stdout")

	if logfile != "" {
		f, err := os.OpenFile(logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
	}

	handler := jsonrpc2.HandlerWithError(handle)
	var connOpt []jsonrpc2.ConnOpt
	<-jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(stdrwc{}, jsonrpc2.VSCodeObjectCodec{}),
		handler, connOpt...).DisconnectNotify()
	log.Println("sqls: connections closed")
}

func handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	switch req.Method {
	case "initialize":
		return handleInitialize(ctx, conn, req)
	case "shutdown":
		return handleShutdown(ctx, conn, req)
		// case "textDocument/didOpen":
		// 	return h.handleTextDocumentDidOpen(ctx, conn, req)
		// case "textDocument/didChange":
		// 	return h.handleTextDocumentDidChange(ctx, conn, req)
		// case "textDocument/didSave":
		// 	return h.handleTextDocumentDidSave(ctx, conn, req)
		// case "textDocument/didClose":
		// 	return h.handleTextDocumentDidClose(ctx, conn, req)
		// case "textDocument/formatting":
		// 	return h.handleTextDocumentFormatting(ctx, conn, req)
		// case "textDocument/documentSymbol":
		// 	return h.handleTextDocumentSymbol(ctx, conn, req)
	}

	return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: fmt.Sprintf("method not supported: %s", req.Method)}
}

type stdrwc struct{}

func (stdrwc) Read(p []byte) (int, error) {
	return os.Stdin.Read(p)
}

func (stdrwc) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

func (stdrwc) Close() error {
	if err := os.Stdin.Close(); err != nil {
		return err
	}
	return os.Stdout.Close()
}

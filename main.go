package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/lighttiger2505/sqls/internal/handler"
)

var (
	logfile string
	trace   bool
)

func main() {
	flag.StringVar(&logfile, "log", "", "also log to this file (in addition to stderr)")
	flag.BoolVar(&trace, "trace", false, "print all requests and responses")
	flag.Parse()

	var logWriter io.Writer
	if logfile != "" {
		f, err := os.OpenFile(logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		logWriter = io.MultiWriter(os.Stderr, f)
	} else {
		logWriter = io.MultiWriter(os.Stderr)
	}
	log.SetOutput(logWriter)

	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(1)
	}

	// Initialize language server
	server := handler.NewServer()
	handler := jsonrpc2.HandlerWithError(server.Handle)

	// Set connect option
	var connOpt []jsonrpc2.ConnOpt
	if trace {
		connOpt = append(connOpt, jsonrpc2.LogMessages(log.New(logWriter, "", 0)))
	}

	// Start language server
	log.Println("sqls: reading on stdin, writing on stdout")
	<-jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(stdrwc{}, jsonrpc2.VSCodeObjectCodec{}),
		handler,
		connOpt...,
	).DisconnectNotify()
	log.Println("sqls: connections closed")
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

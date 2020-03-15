package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/lighttiger2505/sqls/internal/handler"
)

var (
	logfile string
)

func main() {
	flag.StringVar(&logfile, "log", "", "logfile")
	flag.Parse()

	if logfile != "" {
		f, err := os.OpenFile(logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(1)
	}

	if logfile != "" {
		f, err := os.OpenFile(logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
	}

	// Initialize language server
	log.Println("sqls: start service")
	server := handler.NewServer()
	handler := jsonrpc2.HandlerWithError(server.Handle)

	// Start language server
	log.Println("sqls: reading on stdin, writing on stdout")
	var connOpt []jsonrpc2.ConnOpt
	<-jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(stdrwc{}, jsonrpc2.VSCodeObjectCodec{}),
		handler, connOpt...).DisconnectNotify()
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

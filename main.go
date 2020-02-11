package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/lighttiger2505/sqls/database"
	"github.com/sourcegraph/jsonrpc2"
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

	// Create database connection
	db := database.NewMySQLDB("root:root@tcp(127.0.0.1:13306)/world")
	completer := NewCompleter(db)

	// Initialize language server
	log.Println("sqls: start service")
	server := NewServer(completer)
	if err := server.init(); err != nil {
		log.Fatal("sqls: failed database connection, ", err)
	}
	handler := jsonrpc2.HandlerWithError(server.handle)

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

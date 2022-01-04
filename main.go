package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/lighttiger2505/sqls/internal/config"
	"github.com/lighttiger2505/sqls/internal/handler"
)

// builtin variables. see Makefile
var (
	version  string
	revision string
)

var (
	ver        bool
	help       bool
	logfile    string
	trace      bool
	configFile string
)

func main() {
	flag.BoolVar(&help, "help", false, "Print help.")
	flag.BoolVar(&ver, "version", false, "Print version.")
	flag.StringVar(&logfile, "log", "", "Also log to this file. (in addition to stderr)")
	flag.StringVar(&configFile, "config", "", "Specifies an alternative per-user configuration file. If a configuration file is given on the command line, the workspace option (initializationOptions) will be ignored.")
	flag.BoolVar(&trace, "trace", false, "Print all requests and responses.")
	flag.Parse()

	if help {
		fmt.Fprintf(os.Stderr, "usage: sqls [flags]\n")
		flag.PrintDefaults()
		return
	}

	if ver {
		fmt.Fprintf(os.Stderr, "sqls Version:%s, Revision:%s\n", version, revision)
		return
	}

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
	defer func() {
		if err := server.Stop(); err != nil {
			log.Println(err)
		}
	}()
	h := jsonrpc2.HandlerWithError(server.Handle)

	// Load specific config
	if configFile != "" {
		cfg, err := config.GetConfig(configFile)
		if err != nil {
			log.Printf("cannot read specificed config, %+v", err)
		}
		server.SpecificFileCfg = cfg
	} else {
		// Load default config
		cfg, err := config.GetDefaultConfig()
		if err != nil && !errors.Is(config.ErrNotFoundConfig, err) {
			log.Printf("cannot read default config, %+v", err)
		}
		server.DefaultFileCfg = cfg
	}

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
		h,
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

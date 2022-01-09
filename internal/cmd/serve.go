package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/lighttiger2505/sqls/internal/config"
	"github.com/lighttiger2505/sqls/internal/handler"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/urfave/cli/v2"
)

func Serve(c *cli.Context) error {
	logfile := c.String("log")
	configFile := c.String("config")
	trace := c.Bool("trace")
	fmt.Println(logfile, configFile, trace)

	// Initialize log writer
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
			return fmt.Errorf("cannot read specificed config, %w", err)
		}
		server.SpecificFileCfg = cfg
	} else {
		// Load default config
		cfg, err := config.GetDefaultConfig()
		if err != nil && !errors.Is(config.ErrNotFoundConfig, err) {
			return fmt.Errorf("cannot read default config, %w", err)
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

	return nil
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

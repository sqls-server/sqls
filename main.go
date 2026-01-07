package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sourcegraph/jsonrpc2"
	"github.com/urfave/cli/v2"

	"github.com/sqls-server/sqls/internal/config"
	"github.com/sqls-server/sqls/internal/handler"
)

const name = "sqls"

const version = "0.2.45"

var revision = "HEAD"

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

func realMain() error {
	app := &cli.App{
		Name:    name,
		Version: fmt.Sprintf("Version:%s, Revision:%s\n", version, revision),
		Usage:   "An implementation of the Language Server Protocol for SQL.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "log",
				Aliases: []string{"l"},
				Usage:   "Also log to this file. (in addition to stderr)",
			},
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Specifies an alternative per-user configuration file. If a configuration file is given on the command line, the workspace option (initializationOptions) will be ignored.",
			},
			&cli.BoolFlag{
				Name:    "trace",
				Aliases: []string{"t"},
				Usage:   "Print all requests and responses.",
			},
		},
		Commands: cli.Commands{
			{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "edit config",
				Action: func(c *cli.Context) error {
					editorEnv := os.Getenv("EDITOR")
					if editorEnv == "" {
						editorEnv = "vim"
					}
					dir := filepath.Dir(config.YamlConfigPath)
					if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
						if err := os.MkdirAll(dir, 0755); err != nil {
							return fmt.Errorf("cannot create config directory, %w", err)
						}
					}
					return openEditor(editorEnv, config.YamlConfigPath)
				},
			},
		},
		Action: func(c *cli.Context) error {
			return serve(c)
		},
	}
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"v"},
		Usage:   "Print version.",
	}
	cli.HelpFlag = &cli.BoolFlag{
		Name:    "help",
		Aliases: []string{"h"},
		Usage:   "Print help.",
	}

	err := app.Run(os.Args)
	if err != nil {
		return err
	}

	return nil
}

func serve(c *cli.Context) error {
	logfile := c.String("log")
	configFile := c.String("config")
	trace := c.Bool("trace")

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
			return fmt.Errorf("cannot read specified config, %w", err)
		}
		server.SpecificFileCfg = cfg
	} else {
		// Load default config
		cfg, err := config.GetDefaultConfig()
		if err != nil && !errors.Is(err, config.ErrNotFoundConfig) {
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

func openEditor(program string, args ...string) error {
	cmd := exec.CommandContext(context.Background(), program, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

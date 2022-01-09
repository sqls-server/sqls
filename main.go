package main

import (
	"fmt"
	"os"

	"github.com/lighttiger2505/sqls/internal/cmd"
	"github.com/urfave/cli/v2"
)

// builtin variables. see Makefile
var (
	version  string
	revision string
)

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

func realMain() error {
	app := &cli.App{
		Name:    "sqls",
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
				Action:  cmd.Config,
			},
		},
		Action: func(c *cli.Context) error {
			return cmd.Serve(c)
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

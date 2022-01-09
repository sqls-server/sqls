package cmd

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/lighttiger2505/sqls/internal/config"
	"github.com/urfave/cli/v2"
)

func Config(c *cli.Context) error {
	editorEnv := os.Getenv("EDITOR")
	if editorEnv == "" {
		editorEnv = "vim"
	}
	return OpenEditor(editorEnv, config.YamlConfigPath)
}

func OpenEditor(program string, args ...string) error {
	cmdargs := strings.Join(args, " ")
	command := program + " " + cmdargs

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

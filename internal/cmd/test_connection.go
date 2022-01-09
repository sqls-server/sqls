package cmd

import (
	"errors"
	"fmt"

	"github.com/lighttiger2505/sqls/internal/config"
	"github.com/lighttiger2505/sqls/internal/database"
	"github.com/urfave/cli/v2"
)

func TestConnection(c *cli.Context) error {
	configFile := c.String("config")
	var loadedCfg *config.Config

	// Load specific config
	if configFile != "" {
		cfg, err := config.GetConfig(configFile)
		if err != nil {
			return fmt.Errorf("cannot read specificed config, %w", err)
		}
		loadedCfg = cfg
	} else {
		// Load default config
		cfg, err := config.GetDefaultConfig()
		if err != nil && !errors.Is(config.ErrNotFoundConfig, err) {
			return fmt.Errorf("cannot read default config, %w", err)
		}
		loadedCfg = cfg
	}

	// Connect database
	if _, err := database.Open(loadedCfg.Connections[0]); err != nil {
		return fmt.Errorf("Failed connection: %w", err)
	}

	fmt.Println("Success connection")
	return nil
}

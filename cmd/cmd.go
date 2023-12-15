package main

import "github.com/urfave/cli/v2"

const (
	FlagPath      = "path"
	FlagDest      = "dest"
	FlagSource    = "source"
	FlagCompress  = "compress"
	FlagB64Encode = "b64encode"
	FlagLogLevel  = "log-level"
	FlagNamespace = "namespace"
	FlagUseRaw    = "raw"
)

func getCommand() []*cli.Command {
	return []*cli.Command{
		{
			Name:   "backup",
			Usage:  "Run backup",
			Action: backup,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    FlagPath,
					Aliases: []string{"p"},
					Usage:   "List of secrets engine paths to backup",
				},
				&cli.StringFlag{
					Name:    FlagDest,
					Aliases: []string{"d"},
					Usage:   "Local directory to store backup",
					Value:   "backup",
				},
				&cli.BoolFlag{
					Name:    FlagCompress,
					Aliases: []string{"c"},
					Usage:   "Compress backup",
					Value:   false,
				},
				&cli.BoolFlag{
					Name:    FlagB64Encode,
					Aliases: []string{"e"},
					Usage:   "Base64 encode backup values",
					Value:   true,
				},
				&cli.StringFlag{
					Name:    FlagLogLevel,
					Aliases: []string{"l"},
					Usage:   "Log level (debug, info, warn, error, dpanic, panic, fatal)",
					Value:   "info",
				},
				&cli.StringFlag{
					Name:    FlagNamespace,
					Aliases: []string{"n"},
					Usage:   "Vault namespace",
				},
				&cli.BoolFlag{
					Name:    FlagUseRaw,
					Aliases: []string{"r"},
					Usage:   "Use sys/raw endpoint to backup",
				},
			},
		},
		{
			Name:   "restore",
			Usage:  "Run restore",
			Action: restore,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    FlagPath,
					Aliases: []string{"p"},
					Usage:   "Secret engine path to restore to",
				},
				&cli.StringFlag{
					Name:    FlagSource,
					Aliases: []string{"s"},
					Usage:   "Local directory to store backup",
					Value:   "backup",
				},
				&cli.StringFlag{
					Name:    FlagLogLevel,
					Aliases: []string{"l"},
					Usage:   "Log level (debug, info, warn, error, dpanic, panic, fatal)",
					Value:   "info",
				},
				&cli.StringFlag{
					Name:    FlagNamespace,
					Aliases: []string{"n"},
					Usage:   "Vault namespace",
				},
				&cli.BoolFlag{
					Name:    FlagUseRaw,
					Aliases: []string{"r"},
					Usage:   "Use sys/raw endpoint to backup",
				},
			},
		},
	}
}

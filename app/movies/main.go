package movies

import (
	"github.com/urfave/cli/v2"
)

// Movies movie subcomman
func Movies() *cli.Command {
	return &cli.Command{
		Name:        "movies",
		ArgsUsage:   "",
		Usage:       "movies",
		HelpName:    "",
		Description: "Movies subcommand",
		Flags:       []cli.Flag{},
		Subcommands: []*cli.Command{
			Download(),
			List(),
		},
	}
}

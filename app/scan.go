package app

import (
	"fmt"
	"path/filepath"

	"github.com/highercomve/couchness/common"
	"github.com/highercomve/couchness/storage"
	"github.com/urfave/cli/v2"
)

// Scan folder for series and build database
func Scan() *cli.Command {
	return &cli.Command{
		Name:        "scan",
		Aliases:     []string{"s"},
		ArgsUsage:   "",
		Usage:       "scan",
		HelpName:    "",
		Description: "Scan folder for series",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "interactive",
				Aliases: []string{"i"},
			},
			&cli.BoolFlag{
				Name:    "initialize",
				Aliases: []string{"r"},
			},
		},
		Action: func(c *cli.Context) error {
			interactive := c.Bool("interactive")
			initialize := c.Bool("initialize")

			for _, directory := range storage.AppConfiguration.ShowsDirs {
				folderPath, err := filepath.Abs(directory)
				if err != nil {
					return cli.Exit(err.Error(), 0)
				}

				fmt.Println("Scaning folder: " + folderPath)
				shows, err := common.Scan(folderPath+"/", "", interactive, initialize)
				if err != nil {
					return cli.Exit(err.Error(), 0)
				}

				for _, s := range shows {
					fmt.Printf("Show %s with %d episodes in total \n\r", s.Title, len(s.Episodes))
				}
			}

			return nil
		},
	}
}

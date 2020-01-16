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
		Usage:       "scan FOLDER",
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
			args := c.Args()
			folder := args.Get(0)
			interactive := c.Bool("interactive")
			initialize := c.Bool("initialize")

			if folder == "" {
				folder = storage.AppConfiguration.MediaDir
			}

			folderPath, err := filepath.Abs(folder)
			if err != nil {
				return cli.NewExitError(err.Error(), 0)
			}

			fmt.Println("Scaning folder: " + folderPath)
			shows, err := common.Scan(folderPath+"/", interactive, initialize)
			if err != nil {
				return cli.NewExitError(err.Error(), 0)
			}

			for _, s := range shows {
				fmt.Printf("Show %s with %d episodes in total \n\r", s.Title, len(s.Episodes))
			}
			return nil
		},
	}
}

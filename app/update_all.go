package app

import (
	"fmt"
	"path/filepath"

	"github.com/highercomve/couchness/common"
	"github.com/highercomve/couchness/storage"
	"github.com/urfave/cli/v2"
)

// UpdateAll scan your show media and start shows download
func UpdateAll() *cli.Command {
	return &cli.Command{
		Name:        "update-all",
		Aliases:     []string{"ua"},
		ArgsUsage:   "",
		Usage:       "update all your shows",
		HelpName:    "",
		Description: "update all your shows",
		Action: func(c *cli.Context) error {
			fmt.Println("Updating database...")

			for _, directory := range storage.AppConfiguration.MediaDirs {
				folderPath, err := filepath.Abs(directory)
				if err != nil {
					return cli.NewExitError(err.Error(), 0)
				}

				shows, err := common.Scan(folderPath+"/", false, false)
				if err != nil {
					return cli.NewExitError(err.Error(), 0)
				}

				for _, s := range shows {
					fmt.Printf("Updating %s ... \n", s.Title)
					common.Download(s.ID)
				}
			}

			fmt.Printf("\n\r\n\rAll Show now are updated! \n\r")
			return nil
		},
	}
}

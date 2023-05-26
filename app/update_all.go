package app

import (
	"fmt"
	"path/filepath"

	"github.com/highercomve/couchness/common"
	"github.com/highercomve/couchness/models"
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

			for _, directory := range storage.AppConfiguration.ShowsDirs {
				folderPath, err := filepath.Abs(directory)
				if err != nil {
					return cli.Exit(err.Error(), 0)
				}

				shows, err := common.Scan(folderPath+"/", "", false, false)
				if err != nil {
					return cli.Exit(err.Error(), 0)
				}

				for _, s := range shows {
					if s.Configuration.FollowType == models.FollowTypeManual {
						fmt.Printf("Show %s is in manual mode... \n", s.Title)
						continue
					}

					fmt.Printf("Searching for episodes %s ... \n", s.Title)
					common.Download(s.ID)
				}
			}

			fmt.Printf("\n\r\n\rAll Show now are updated! \n\r")
			return nil
		},
	}
}

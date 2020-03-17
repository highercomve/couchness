package app

import (
	"fmt"

	"github.com/highercomve/couchness/common"
	"github.com/urfave/cli/v2"
)

// AddMedia add new media directory to database
func AddMedia() *cli.Command {
	return &cli.Command{
		Name:        "add-media",
		Aliases:     []string{"am"},
		ArgsUsage:   "",
		Usage:       "add-media directory",
		Description: "Add a new media directory to scan and follow",
		Flags:       []cli.Flag{},
		Action: func(c *cli.Context) error {
			args := c.Args()
			directory := args.Get(0)

			fmt.Printf("Adding new media directory: %s \n\r", directory)

			err := common.AddMedia(directory)
			if err != nil {
				return cli.NewExitError(err.Error(), 0)
			}

			fmt.Printf("Media directory added: %s \n\r", directory)
			return nil
		},
	}
}

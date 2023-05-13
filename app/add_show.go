package app

import (
	"fmt"

	"github.com/highercomve/couchness/common"
	"github.com/urfave/cli/v2"
)

// AddShowsDirectory add new media directory to database
func AddShowsDirectory() *cli.Command {
	return &cli.Command{
		Name:        "add-shows-dir",
		Aliases:     []string{"asd"},
		ArgsUsage:   "",
		Usage:       "add-shows-dir <directory>",
		Description: "Add a new shows directory to scan and follow",
		Flags:       []cli.Flag{},
		Action: func(c *cli.Context) error {
			args := c.Args()
			directory := args.Get(0)

			fmt.Printf("Adding new media directory: %s \n\r", directory)

			err := common.AddMedia(directory)
			if err != nil {
				return cli.Exit(err.Error(), 0)
			}

			fmt.Printf("Media directory added: %s \n\r", directory)
			return nil
		},
	}
}

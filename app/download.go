package app

import (
	"fmt"
	"github.com/highercomve/couchness/common"
	"github.com/urfave/cli/v2"
)

// Download folder for series and build database
func Download() *cli.Command {
	return &cli.Command{
		Name:        "download",
		Aliases:     []string{"d"},
		ArgsUsage:   "showID",
		Usage:       "",
		HelpName:    "",
		Description: "Download show by ID",
		Flags:       []cli.Flag{},
		Action: func(c *cli.Context) error {
			args := c.Args()
			showID := args.Get(0)

			if showID == "" {
				return cli.NewExitError("the showID is requeried", 0)
			}

			fmt.Println("Starting posible download of " + showID)

			err := common.Download(showID)
			if err != nil {
				fmt.Printf("%s \n", err.Error())
				return nil
			}

			fmt.Printf("Show %s is now in transmission download queue \n", showID)
			return nil
		},
	}
}

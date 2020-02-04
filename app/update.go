package app

import (
	"fmt"

	"github.com/highercomve/couchness/common"
	"github.com/urfave/cli/v2"
)

// Update scan your show media and start shows download
func Update() *cli.Command {
	return &cli.Command{
		Name:        "update",
		Aliases:     []string{"u"},
		ArgsUsage:   "",
		Usage:       "update one show using showID",
		HelpName:    "",
		Description: "Scan and download",
		Action: func(c *cli.Context) error {
			args := c.Args()
			showID := args.Get(0)

			if showID == "" {
				return cli.NewExitError("the showID is requeried", 0)
			}

			fmt.Println("Scanning and updating " + showID + " ...")

			err := common.Update(showID)
			if err != nil {
				return cli.NewExitError(err.Error(), 0)
			}

			fmt.Printf("\n\r\n\rAll Show now are updated! \n\r")
			return nil
		},
	}
}

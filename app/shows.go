package app

import (
	"encoding/json"
	"fmt"

	"github.com/highercomve/couchness/common"
	"github.com/urfave/cli/v2"
)

// Shows scan your show media and start shows download
func Shows() *cli.Command {
	return &cli.Command{
		Name:        "shows",
		ArgsUsage:   "",
		Usage:       "show [SHOW_ID]",
		HelpName:    "",
		Description: "List all or one show",
		Action: func(c *cli.Context) error {
			args := c.Args()
			showID := args.Get(0)

			shows, err := common.GetShow(showID)
			if err != nil {
				return cli.NewExitError(err.Error(), 0)
			}

			showJSON, _ := json.MarshalIndent(shows, "", "  ")
			fmt.Printf("%s \n", showJSON)

			return nil
		},
	}
}

package app

import (
	"encoding/json"
	"fmt"

	"github.com/highercomve/couchness/common"
	"github.com/urfave/cli/v2"
)

// DisableShow disable auto update of show
func DisableShow() *cli.Command {
	return &cli.Command{
		Name:        "disable",
		ArgsUsage:   "",
		Usage:       "disable [SHOW_ID]",
		HelpName:    "",
		Description: "Disable auto update on one show",
		Flags:       []cli.Flag{},
		Action: func(c *cli.Context) error {
			args := c.Args()
			showID := args.Get(0)

			if showID == "" {
				return cli.Exit("Show id is need it", 0)
			}

			shows, err := common.DisableShow(showID)
			if err != nil {
				return cli.Exit(err.Error(), 0)
			}

			showJSON, _ := json.MarshalIndent(shows, "", "  ")
			fmt.Printf("%s \n", showJSON)

			return nil
		},
	}
}

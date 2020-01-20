package app

import (
	"encoding/json"
	"fmt"

	"github.com/highercomve/couchness/common"
	"github.com/urfave/cli/v2"
)

// Show scan your show media and start shows ownload
func Show() *cli.Command {
	return &cli.Command{
		Name:        "show",
		ArgsUsage:   "",
		Usage:       "show [SHOW_ID]",
		HelpName:    "",
		Description: "Details one show",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "detail",
				Aliases: []string{"d"},
			},
		},
		Action: func(c *cli.Context) error {
			args := c.Args()
			showID := args.Get(0)
			detail := c.Bool("detail")

			if showID == "" {
				return cli.NewExitError("Show id is need it", 0)
			}

			shows, err := common.GetShow(showID, detail)
			if err != nil {
				return cli.NewExitError(err.Error(), 0)
			}
			showJSON, _ := json.MarshalIndent(shows, "", "  ")
			fmt.Printf("%s \n", showJSON)

			return nil
		},
	}
}

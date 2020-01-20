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
		Usage:       "show",
		HelpName:    "",
		Description: "List all or one show",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "detail",
				Aliases: []string{"d"},
			},
		},
		Action: func(c *cli.Context) error {
			detail := c.Bool("detail")

			shows, err := common.GetShows()
			if err != nil {
				return cli.NewExitError(err.Error(), 0)
			}

			if detail {
				showJSON, _ := json.MarshalIndent(shows, "", "  ")
				fmt.Printf("%s \n", showJSON)
				return nil
			}

			for _, s := range shows {
				fmt.Println(s.Summary())
			}

			return nil
		},
	}
}

package app

import (
	"encoding/json"
	"fmt"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
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

			if showID == "" {
				shows, err := storage.GetAllShows()
				if err != nil {
					return cli.NewExitError(err.Error(), 0)
				}
				for _, s := range shows {
					countEpisodes(s)
					shows = append(shows, s)
				}
				showsJSON, _ := json.MarshalIndent(shows, "", "  ")
				fmt.Printf("%s \n", showsJSON)

				return nil
			}

			show := &models.Show{}
			err := storage.Db.Driver.Read(storage.Db.Collections.Shows, showID, show)
			if err != nil {
				return cli.NewExitError(err.Error(), 0)
			}

			showJSON, _ := json.MarshalIndent(show, "", "  ")
			fmt.Printf("%s \n", showJSON)

			return nil
		},
	}
}

func countEpisodes(s *models.Show) {
	s.EpisodesCount = len(s.Episodes)
	s.Episodes = make(models.Episodes, 0)
}

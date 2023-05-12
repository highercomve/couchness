package app

import (
	"fmt"
	"strconv"

	"github.com/highercomve/couchness/common"
	"github.com/urfave/cli/v2"
)

// Download folder for series and build database
func Download() *cli.Command {
	return &cli.Command{
		Name:        "download",
		Aliases:     []string{"d"},
		ArgsUsage:   "showID",
		Usage:       "download SHOW_ID",
		HelpName:    "",
		Description: "Download show by ID",
		Flags:       []cli.Flag{},
		Action: func(c *cli.Context) error {
			args := c.Args()
			showID := args.Get(0)

			if showID == "" {
				return cli.Exit("the showID is requeried", 0)
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

// DownloadEpisode download specific episode of a show
func DownloadEpisode() *cli.Command {
	return &cli.Command{
		Name:        "download-ep",
		Aliases:     []string{"de"},
		ArgsUsage:   "showID",
		Usage:       "download SHOW_ID EPISODE maximun_search(optional)",
		HelpName:    "",
		Description: "Download show by ID and episode",
		Flags:       []cli.Flag{},
		Action: func(c *cli.Context) error {
			args := c.Args()
			showID := args.Get(0)
			episode := args.Get(1)
			max := args.Get(2)
			limit := 100

			if showID == "" {
				return cli.Exit("the showID is required", 0)
			}
			if episode == "" {
				return cli.Exit("the episode is required", 0)
			}

			if max != "" {
				if m, err := strconv.Atoi(max); err == nil {
					limit = m
				}
			}

			fmt.Println("Searching Episode on show services \n\r")

			err := common.DownloadEpisode(showID, episode, limit)
			if err != nil {
				fmt.Printf("%s \n", err.Error())
				return nil
			}

			fmt.Printf("Show %s is now in transmission download queue \n", showID)
			return nil
		},
	}
}

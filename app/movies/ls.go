package movies

import (
	"fmt"

	"github.com/highercomve/couchness/common"
	"github.com/urfave/cli/v2"
)

// List show all movies in db
func List() *cli.Command {
	return &cli.Command{
		Name:        "ls",
		ArgsUsage:   "",
		Usage:       "ls",
		HelpName:    "",
		Description: "Show list of all movies",
		Flags:       []cli.Flag{},
		Action: func(c *cli.Context) error {
			movies, err := common.GetMovies()
			if err != nil {
				return cli.Exit(err.Error(), 0)
			}

			for _, movie := range movies {
				fmt.Println(movie.Summary())
			}

			return nil
		},
	}
}

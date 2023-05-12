package app

import (
	"github.com/highercomve/couchness/services/eztv"
	"github.com/highercomve/couchness/services/rarbg"
	"github.com/highercomve/couchness/storage"
	"github.com/urfave/cli/v2"
)

// Migrate add new series to follow
func Migrate() *cli.Command {
	return &cli.Command{
		Name:        "migrate",
		Aliases:     []string{"m"},
		ArgsUsage:   "",
		Usage:       "Migrate shows from monoservice to multiservice",
		Description: "Get all your shows with service",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        "services",
				Aliases:     []string{"s"},
				Usage:       "Coma separated type of services (showrss, eztv, rarbg)",
				DefaultText: "[eztv, rarbg]",
				Required:    false,
			},
		},
		Action: func(c *cli.Context) error {
			services := c.StringSlice("services")
			if len(services) == 0 {
				services = []string{eztv.ServiceType, rarbg.ServiceType}
			}

			shows, err := storage.GetAllShows()
			if err != nil {
				return cli.Exit(err.Error(), 0)
			}

			for _, show := range shows {
				if len(show.Configuration.Services) == 0 {
					show.Configuration.Services = services
					storage.NewShowStorage(show).Save()
				}
			}

			return nil
		},
	}
}

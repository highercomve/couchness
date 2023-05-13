package movies

import (
	"fmt"
	"path/filepath"

	"github.com/gosimple/slug"
	"github.com/highercomve/couchness/common"
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/services/eztv"
	"github.com/highercomve/couchness/services/rarbg"
	"github.com/highercomve/couchness/storage"
	"github.com/urfave/cli/v2"
)

// Download add a new movie
func Download() *cli.Command {
	return &cli.Command{
		Name:        "download",
		ArgsUsage:   "",
		Usage:       "download <movie> <folder>",
		Description: "download a new movie",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "title",
				Aliases:  []string{"n"},
				Usage:    "movie identification name",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "resolution",
				Aliases:  []string{"r"},
				Usage:    "Resolution of the torrent",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "codec",
				Aliases:  []string{"c"},
				Usage:    "Codec to be downloaded",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "quality",
				Aliases:  []string{"q"},
				Usage:    "Quality of the torrent",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "external-id",
				Aliases:  []string{"ex"},
				Usage:    "assign external ID",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "dir",
				Aliases:  []string{"d"},
				Usage:    "movie directory name",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "folder",
				Aliases:  []string{"f"},
				Usage:    "Where is going to save the show",
				Required: false,
			},
			&cli.StringSliceFlag{
				Name:        "services",
				Aliases:     []string{"s"},
				Usage:       "Coma separated type of services (showrss, eztv, rarbg)",
				DefaultText: "[eztv, rarbg]",
				Required:    false,
			},
		},
		Action: func(c *cli.Context) error {
			key := c.String("key")
			title := c.String("title")
			folder := c.String("folder")
			services := c.StringSlice("services")
			externalID := c.String("external-id")
			codec := c.String("codec")
			quality := c.String("quality")
			resolution := c.String("resolution")

			if key == "" {
				key = slug.Make(title)
			}

			if len(services) == 0 {
				services = []string{eztv.ServiceType, rarbg.ServiceType}
			}

			if folder == "" {
				folder = storage.AppConfiguration.MoviesDir
			}

			folderPath, err := filepath.Abs(folder)
			if err != nil {
				return cli.Exit(err.Error(), 0)
			}

			if externalID == "" {
				title, key, externalID, err = common.SearchAndSelectOnImdb(title, "movie")
				if err != nil {
					return cli.Exit(err.Error(), 0)
				}
			}

			movie := &models.Movie{
				Show: models.Show{
					ID:         key,
					Title:      title,
					Directory:  folderPath + "/",
					ExternalID: externalID,
					Configuration: &models.ShowConf{
						Services:   services,
						Codec:      codec,
						Quality:    quality,
						Resolution: resolution,
					},
				},
			}

			_, err = common.AddMovie(movie)
			if err != nil {
				return cli.Exit(err.Error(), 0)
			}

			fmt.Printf("\n\r\n\r Movie schedule to download at: %s \n\r", movie.Directory)
			return nil
		},
	}
}

package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/gosimple/slug"
	"github.com/highercomve/couchness/common"
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
	"github.com/urfave/cli/v2"
)

// Add add new series to follow
func Add() *cli.Command {
	return &cli.Command{
		Name:        "add",
		Aliases:     []string{"a"},
		ArgsUsage:   "",
		Usage:       "add SHOW_NAME FOLDER",
		Description: "Add new show to follow",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "title",
				Aliases:  []string{"n"},
				Usage:    "Show identification name",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "resolution",
				Aliases:  []string{"r"},
				Usage:    "Resolution of the torrent",
				Required: true,
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
				Name:     "key",
				Aliases:  []string{"k"},
				Usage:    "Show directory name",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "folder",
				Aliases:  []string{"f"},
				Usage:    "Where is going to save the show",
				Required: false,
			},
			&cli.StringFlag{
				Name:        "service",
				Aliases:     []string{"s"},
				Usage:       "Type of service (showrss, eztv)",
				DefaultText: "eztv",
				Required:    false,
			},
			&cli.StringFlag{
				Name:        "type",
				Aliases:     []string{"t"},
				Usage:       "Type of show sync (latest, since, all)",
				DefaultText: "latest",
				Required:    false,
			},
			&cli.IntFlag{
				Name:     "season",
				Usage:    "since seasson",
				Required: false,
			},
		},
		Action: func(c *cli.Context) error {
			key := c.String("key")
			title := c.String("title")
			folder := c.String("folder")
			service := c.String("service")
			followType := c.String("type")
			externalID := c.String("external-id")
			codec := c.String("codec")
			quality := c.String("quality")
			resolution := c.String("resolution")
			season := c.Int("season")

			if key == "" {
				key = slug.Make(title)
			}

			if service == "" {
				service = "eztv"
			}

			if followType == "" {
				followType = "latest"
			}

			if folder == "" {
				folder = storage.AppConfiguration.MediaDir + key
			}

			folderPath, err := filepath.Abs(folder)
			if err != nil {
				return cli.NewExitError(err.Error(), 0)
			}

			if externalID == "" {
				title, key, externalID, err = common.SearchAndSelectOnImdb(title)
				if err != nil {
					return cli.NewExitError(err.Error(), 0)
				}
			}

			show := &models.Show{
				ID:         key,
				Title:      title,
				Directory:  folderPath + "/",
				ExternalID: externalID,
				Configuration: &models.ShowConf{
					Service:    service,
					FollowType: followType,
					Codec:      codec,
					Quality:    quality,
					Resolution: resolution,
					Since:      season,
				},
			}

			fmt.Printf("Adding show %s \n\r", show.PrintString())

			_, err = common.Add(show)
			if err != nil {
				return cli.NewExitError(err.Error(), 0)
			}

			showData, _ := json.MarshalIndent(show, "", "    ")

			fmt.Printf("%s \n", showData)
			fmt.Printf("saved on: %s/%s/%s.json \n\r", storage.DbDir, storage.Db.Collections.Shows, key)
			return nil
		},
	}
}

package main

import (
	"os"

	"github.com/highercomve/couchness/app"
	storage "github.com/highercomve/couchness/storage"
	cli "github.com/urfave/cli/v2"
)

var externalCommands []*cli.Command = []*cli.Command{}

func main() {
	a := &cli.App{
		Before: func(c *cli.Context) error {
			configDir := c.Path("config-dir")
			return storage.Init(configDir)
		},
		EnableBashCompletion: true,
		Name:                 "couchness",
		HelpName:             "couchness",
		Usage:                "couchness is an automatic tool to follow and download show using RSS or eztv",
		Version:              Version,
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name: "config-dir",
			},
		},
		Authors: []*cli.Author{
			{
				Name:  "Sergio Marin",
				Email: "",
			},
		},
	}
	a.Commands = append(app.Commands, externalCommands...)
	a.Run(os.Args)
}

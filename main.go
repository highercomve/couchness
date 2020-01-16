package main

import (
	"log"
	"os"

	"github.com/highercomve/couchness/app"
	storage "github.com/highercomve/couchness/storage"
	cli "github.com/urfave/cli/v2"
)

var externalCommands []*cli.Command = []*cli.Command{}

func main() {
	a := &cli.App{
		EnableBashCompletion: true,
		Name:                 "couchness",
		HelpName:             "couchness",
		Usage:                "couchness is an automatic tool to follow and download show using RSS or eztv",
		Version:              Version,
		Authors: []*cli.Author{
			&cli.Author{
				Name:  "Sergio Marin",
				Email: "",
			},
		},
	}

	a.Commands = append(app.Commands, externalCommands...)
	err := storage.Init()
	if err != nil {
		log.Fatal(err)
	}
	a.Run(os.Args)
}

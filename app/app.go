package app

import "github.com/urfave/cli/v2"

// Commands list of all commands
var Commands []*cli.Command = []*cli.Command{
	Add(),
	Scan(),
	Download(),
	UpdateAll(),
	Migrate(),
	Update(),
	Shows(),
	Show(),
	AddMedia(),
}

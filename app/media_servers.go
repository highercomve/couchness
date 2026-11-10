package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/services/mediaserver"
	"github.com/highercomve/couchness/storage"
	cli "github.com/urfave/cli/v2"
	"golang.org/x/term"
)

// MediaServers manages the Jellyfin/Emby servers couchness asks to rescan.
func MediaServers() *cli.Command {
	return &cli.Command{
		Name:  "media-servers",
		Usage: "configure, test and rescan the Jellyfin/Emby servers that show your library",
		Subcommands: []*cli.Command{
			mediaServersList(),
			mediaServersSet(),
			mediaServersRemove(),
			mediaServersTest(),
			mediaServersRefresh(),
		},
	}
}

func validKind(kind string) (string, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	switch kind {
	case models.MediaServerJellyfin, models.MediaServerEmby:
		return kind, nil
	}
	return "", fmt.Errorf("unknown media server %q: use jellyfin or emby", kind)
}

func mediaServersList() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "show the configured servers (passwords are never printed)",
		Action: func(c *cli.Context) error {
			if len(storage.AppConfiguration.MediaServers) == 0 {
				fmt.Println("No media servers configured. Add one with: couchness media-servers set jellyfin --url http://localhost:8097 --username USER --password-stdin")
				return nil
			}
			for _, m := range storage.AppConfiguration.MediaServers {
				auth := "not configured"
				switch {
				case m.APIKey != "":
					auth = "API key"
				case m.Username != "":
					auth = "user " + m.Username
				}
				fmt.Printf("%-9s %s  (%s)\n", m.Type, m.URL, auth)
			}
			return nil
		},
	}
}

func mediaServersSet() *cli.Command {
	// "jellyfin" and "emby" are subcommands rather than an argument: the CLI
	// library stops reading flags at the first plain argument, so
	// "set jellyfin --url ..." would otherwise ignore every flag.
	return &cli.Command{
		Name:  "set",
		Usage: "set the URL and credentials of a jellyfin or emby server",
		// also accepted before the server name: set --url ... jellyfin
		Flags: setFlags("http://localhost:8097"),
		Subcommands: []*cli.Command{
			mediaServerSetKind(models.MediaServerJellyfin, "http://localhost:8097"),
			mediaServerSetKind(models.MediaServerEmby, "http://localhost:8096"),
		},
	}
}

func mediaServerSetKind(kind, example string) *cli.Command {
	return &cli.Command{
		Name:  kind,
		Usage: "set the " + kind + " server, e.g. --url " + example + " --username USER --password-stdin",
		Flags: setFlags(example),
		Action: func(c *cli.Context) error {
			password := flagString(c, "password")
			if flagBool(c, "password-stdin") {
				fmt.Fprint(os.Stderr, "Password: ")
				line, err := readPassword()
				fmt.Fprintln(os.Stderr)
				if err != nil {
					return cli.Exit("could not read the password from standard input", 1)
				}
				password = line
			}
			server := models.MediaServer{
				Type: kind, URL: strings.TrimRight(strings.TrimSpace(flagString(c, "url")), "/"),
				Username: strings.TrimSpace(flagString(c, "username")), Password: password,
				APIKey: strings.TrimSpace(flagString(c, "api-key")),
			}
			if server.URL == "" {
				return cli.Exit("--url is required, e.g. --url "+example, 1)
			}
			if !server.Configured() {
				return cli.Exit("give --username (with a password) or --api-key", 1)
			}
			cfg := *storage.AppConfiguration
			cfg.SetMediaServer(server)
			if err := storage.Db.SaveAppConfiguration(&cfg); err != nil {
				return cli.Exit(err.Error(), 1)
			}
			*storage.AppConfiguration = cfg
			fmt.Printf("Saved %s settings. Checking the connection...\n", kind)
			return testServer(server)
		},
	}
}

// setFlags are defined on "set" and on "set jellyfin|emby"; the CLI library
// looks values up through parent commands, so both orders work.
func setFlags(example string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "url", Usage: "server address, e.g. " + example},
		&cli.StringFlag{Name: "username", Usage: "user to sign in as (needs to be an administrator to rescan)"},
		&cli.BoolFlag{Name: "password-stdin", Usage: "read the password from standard input instead of the command line"},
		&cli.StringFlag{Name: "password", Usage: "password (prefer --password-stdin: arguments end up in shell history)"},
		&cli.StringFlag{Name: "api-key", Usage: "API key (Dashboard -> API Keys); used instead of username/password"},
	}
}

// flagString returns a flag from the command or, when not set there, from
// a parent command ("set --url X jellyfin").
func flagString(c *cli.Context, name string) string {
	for _, ctx := range c.Lineage() {
		if ctx.IsSet(name) {
			return ctx.String(name)
		}
	}
	return ""
}

func flagBool(c *cli.Context, name string) bool {
	for _, ctx := range c.Lineage() {
		if ctx.IsSet(name) {
			return ctx.Bool(name)
		}
	}
	return false
}

// readPassword reads one line from stdin, hiding the typing when stdin is a
// terminal.
func readPassword() (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		return string(b), err
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func mediaServersRemove() *cli.Command {
	return &cli.Command{
		Name:      "remove",
		Usage:     "forget a jellyfin or emby server",
		ArgsUsage: "jellyfin|emby",
		Action: func(c *cli.Context) error {
			kind, err := validKind(c.Args().First())
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			cfg := *storage.AppConfiguration
			cfg.SetMediaServer(models.MediaServer{Type: kind})
			if err := storage.Db.SaveAppConfiguration(&cfg); err != nil {
				return cli.Exit(err.Error(), 1)
			}
			*storage.AppConfiguration = cfg
			fmt.Printf("Removed %s settings.\n", kind)
			return nil
		},
	}
}

func mediaServersTest() *cli.Command {
	return &cli.Command{
		Name:  "test",
		Usage: "sign in to every configured server and show its name and version",
		Action: func(c *cli.Context) error {
			servers := configuredServers()
			if len(servers) == 0 {
				return cli.Exit("no media servers configured", 1)
			}
			failed := false
			for _, s := range servers {
				if testServer(s) != nil {
					failed = true
				}
			}
			if failed {
				return cli.Exit("", 1)
			}
			return nil
		},
	}
}

func mediaServersRefresh() *cli.Command {
	return &cli.Command{
		Name:  "refresh",
		Usage: "ask every configured server to rescan its libraries",
		Action: func(c *cli.Context) error {
			servers := configuredServers()
			if len(servers) == 0 {
				return cli.Exit("no media servers configured", 1)
			}
			if !ReportRefresh(mediaserver.RefreshAll(servers)) {
				return cli.Exit("", 1)
			}
			return nil
		},
	}
}

func configuredServers() []models.MediaServer {
	var out []models.MediaServer
	for _, s := range storage.AppConfiguration.MediaServers {
		if s.Configured() {
			out = append(out, s)
		}
	}
	return out
}

func testServer(s models.MediaServer) error {
	client := mediaserver.New(s)
	info, err := client.Info()
	if err != nil {
		fmt.Printf("✗ %s: %v\n", client.Name(), err)
		return err
	}
	fmt.Printf("✓ %s: signed in to %q, version %s\n", client.Name(), info.ServerName, info.Version)
	return nil
}

// ReportRefresh prints refresh results and reports whether all succeeded.
func ReportRefresh(results []mediaserver.Result) bool {
	ok := true
	for _, r := range results {
		if r.Err != nil {
			ok = false
			fmt.Printf("✗ %s: rescan failed: %v\n", r.Name, r.Err)
			continue
		}
		fmt.Printf("✓ %s: library rescan started\n", r.Name)
	}
	return ok
}

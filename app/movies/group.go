package movies

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/highercomve/couchness/common"
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/services/mediaserver"
	"github.com/highercomve/couchness/storage"
	"github.com/highercomve/couchness/utils/humanize"
	"github.com/urfave/cli/v2"
)

// Group moves copies of the same movie into one folder so Jellyfin, Emby
// and Plex show a single movie with a version picker.
func Group() *cli.Command {
	return &cli.Command{
		Name:    "group",
		Aliases: []string{"organize"},
		Usage:   "group copies of the same movie as versions (Jellyfin/Emby/Plex layout)",
		Description: `Finds movies stored more than once in the movies folder and moves every copy into
"Title (Year)/Title (Year) - <label>.<ext>", the layout Jellyfin uses to show one movie with
several versions. Subtitles, .nfo files, extras folders and artwork move with their video.

Without --apply it only prints the plan. Files are never deleted or overwritten; folders are
removed only when they end up empty. Movies still held by transmission are skipped unless
--force is given, because moving them would break seeding.`,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "dir", Usage: "movies folder (default: movies_dir from the configuration)"},
			&cli.BoolFlag{Name: "apply", Usage: "move the files (default is a dry run)"},
			&cli.BoolFlag{Name: "force", Usage: "also move movies that transmission still holds"},
			&cli.Int64Flag{Name: "min-size", Value: common.DefaultMinMovieSize >> 20, Usage: "smallest file in MiB treated as a movie"},
			&cli.StringFlag{Name: "jellyfin-url", EnvVars: []string{"COUCHNESS_JELLYFIN_URL"}, Usage: "one-off Jellyfin URL to rescan (default: servers from `couchness media-servers`)"},
			&cli.StringFlag{Name: "jellyfin-key", EnvVars: []string{"COUCHNESS_JELLYFIN_API_KEY"}, Usage: "API key for --jellyfin-url"},
		},
		Action: func(c *cli.Context) error {
			dir := c.String("dir")
			if dir == "" {
				dir = storage.AppConfiguration.MoviesDir
			}
			if dir == "" {
				return cli.Exit("no movies folder: pass --dir or set movies_dir in the configuration", 1)
			}

			opts := common.GroupOptions{MinSize: c.Int64("min-size") << 20, Force: c.Bool("force")}
			torrents, err := common.TorrentPaths()
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not ask transmission which movies are still seeding: %v\n", err)
				if c.Bool("apply") && !c.Bool("force") {
					return cli.Exit("refusing to move files without knowing what transmission is seeding; fix the connection or use --force", 1)
				}
			} else {
				opts.Busy = busyChecker(torrents)
			}

			plan, err := common.PlanMovieGroups(dir, opts)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			printPlan(plan)

			moves := plan.Moves()
			if len(moves) == 0 {
				fmt.Println("\nNothing to move.")
				return nil
			}
			if !c.Bool("apply") {
				fmt.Printf("\nDry run: %d file(s) would move; nothing was changed. Run again with --apply to move them.\n", len(moves))
				return nil
			}

			fmt.Println("\nMoving files...")
			res := common.ApplyMovieGroups(plan, func(format string, args ...any) { fmt.Printf(format+"\n", args...) })
			fmt.Printf("\nMoved %d file(s), %d failed, removed %d empty folder(s).\n", res.Moved, len(res.Failed), len(res.RemovedDir))
			for _, e := range res.Failed {
				fmt.Fprintf(os.Stderr, "  failed: %v\n", e)
			}
			if res.Moved > 0 {
				refreshMediaServers(c.String("jellyfin-url"), c.String("jellyfin-key"))
			}
			if len(res.Failed) > 0 {
				return cli.Exit("some files could not be moved", 1)
			}
			return nil
		},
	}
}

// busyChecker reports whether path is a torrent's data or lies inside it.
func busyChecker(torrents map[string]string) func(string) (string, bool) {
	return func(path string) (string, bool) {
		path = filepath.Clean(path)
		for p, name := range torrents {
			if path == p || strings.HasPrefix(path, p+string(os.PathSeparator)) {
				return name, true
			}
		}
		return "", false
	}
}

func printPlan(plan *common.GroupPlan) {
	rel := func(p string) string {
		if r, err := filepath.Rel(plan.Root, p); err == nil {
			return r
		}
		return p
	}
	if len(plan.Groups) == 0 {
		fmt.Printf("No movie is stored more than once in %s.\n", plan.Root)
		return
	}
	fmt.Printf("Movies stored more than once in %s:\n", plan.Root)
	for _, g := range plan.Groups {
		fmt.Printf("\n== %s\n", filepath.Base(g.Folder))
		if g.Skip != "" {
			fmt.Printf("   SKIPPED: %s\n", g.Skip)
			for _, v := range g.Versions {
				fmt.Printf("   - %s\n", rel(v.Path))
			}
			continue
		}
		for _, v := range g.Versions {
			fmt.Printf("   [%s] %s\n", humanize.Bytes(uint64(v.Size)), v.Label)
			if v.Path == v.Dest {
				fmt.Printf("        already in place\n")
				continue
			}
			fmt.Printf("        from %s\n", rel(v.Path))
			fmt.Printf("          to %s\n", rel(v.Dest))
			moving := 0
			for _, m := range v.Companions {
				if m.From != m.To {
					moving++
				}
			}
			if moving > 0 {
				fmt.Printf("        + %d subtitle/extra file(s) move along\n", moving)
			}
			if len(v.Leftovers) > 0 {
				names := make([]string, 0, len(v.Leftovers))
				for _, l := range v.Leftovers {
					names = append(names, filepath.Base(l))
				}
				fmt.Printf("        stays behind (not deleted): %s\n", strings.Join(names, ", "))
			}
		}
	}
}

// refreshMediaServers rescans the servers from the configuration, or the one
// given with --jellyfin-url/--jellyfin-key for a one-off run.
func refreshMediaServers(url, key string) {
	servers := storage.AppConfiguration.MediaServers
	if url != "" && key != "" {
		servers = []models.MediaServer{{Type: models.MediaServerJellyfin, URL: url, APIKey: key}}
	}
	results := mediaserver.RefreshAll(servers)
	if len(results) == 0 {
		fmt.Println("Tip: let couchness rescan Jellyfin/Emby for you: couchness media-servers set jellyfin --url http://localhost:8097 --username USER --password-stdin")
		return
	}
	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: rescan failed: %v\n", r.Name, r.Err)
			continue
		}
		fmt.Printf("✓ %s: library rescan started\n", r.Name)
	}
}

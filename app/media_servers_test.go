package app

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
	cli "github.com/urfave/cli/v2"
)

// The flags come after "jellyfin", exactly as users type it.
func TestMediaServersSetParsesFlagsAfterKind(t *testing.T) {
	for name, args := range map[string][]string{
		"flags after jellyfin":  {"couchness", "media-servers", "set", "jellyfin", "--url", "URL/", "--username", "highercomve", "--password", "pw"},
		"flags before jellyfin": {"couchness", "media-servers", "set", "--url", "URL/", "--username", "highercomve", "--password", "pw", "jellyfin"},
	} {
		t.Run(name, func(t *testing.T) { runMediaServersSet(t, args) })
	}
}

func runMediaServersSet(t *testing.T, args []string) {
	jellyfin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/Users/AuthenticateByName":
			_, _ = w.Write([]byte(`{"AccessToken":"tok"}`))
		case "/System/Info":
			_, _ = w.Write([]byte(`{"ServerName":"rockpi-4b","Version":"10.11.11"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer jellyfin.Close()

	originalDB, originalConfig := storage.Db, storage.AppConfiguration
	t.Cleanup(func() { storage.Db, storage.AppConfiguration = originalDB, originalConfig })
	db, err := storage.New(filepath.Join(t.TempDir(), "db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	storage.Db = db
	storage.AppConfiguration = &models.AppConfiguration{}

	app := &cli.App{Commands: []*cli.Command{MediaServers()}, ExitErrHandler: func(*cli.Context, error) {}}
	for i, a := range args {
		if a == "URL/" {
			args[i] = jellyfin.URL + "/"
		}
	}
	err = app.Run(args)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := storage.AppConfiguration.MediaServer(models.MediaServerJellyfin)
	if !ok || got.URL != jellyfin.URL || got.Username != "highercomve" || got.Password != "pw" {
		t.Fatalf("not saved correctly: %+v", got)
	}
	persisted, err := storage.Db.GetAppConfiguration(&models.AppConfiguration{})
	if err != nil {
		t.Fatal(err)
	}
	if m, ok := persisted.MediaServer(models.MediaServerJellyfin); !ok || m.Password != "pw" {
		t.Fatalf("not persisted: %+v", persisted.MediaServers)
	}
}

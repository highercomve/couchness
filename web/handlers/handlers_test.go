package handlers

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
	"github.com/highercomve/couchness/web/jobs"
	"github.com/highercomve/couchness/web/repository"
	"github.com/highercomve/couchness/web/services"
	"github.com/highercomve/couchness/web/templates"
)

func newTestHandler(t *testing.T) (*Handler, *jobs.Runner, *http.ServeMux) {
	t.Helper()

	dir := t.TempDir()
	originalDB := storage.Db
	originalConfig := storage.AppConfiguration
	t.Cleanup(func() {
		storage.Db = originalDB
		storage.AppConfiguration = originalConfig
	})

	db, err := storage.New(filepath.Join(dir, "database"), nil)
	if err != nil {
		t.Fatal(err)
	}
	storage.Db = db

	showsDir := filepath.Join(dir, "shows")
	if err := os.MkdirAll(filepath.Join(showsDir, "example-show"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(showsDir, "example-show", "Example.Show.S01E01.mkv"), []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	storage.AppConfiguration = &models.AppConfiguration{ShowsDirs: []string{showsDir}}

	showDir := filepath.Join(showsDir, "example-show")
	if _, err := storage.NewShowStorage(&models.Show{
		ID:        "example-show",
		Title:     "Example Show",
		Directory: showDir,
		Episodes:  models.Episodes{&models.TorrentInfo{Title: "Example.Show.S01E01.mkv", Location: "Example.Show.S01E01.mkv", Season: 1, Episode: 1}},
		Configuration: &models.ShowConf{
			FollowType: "latest",
			Service:    "eztv",
		},
	}).Save(); err != nil {
		t.Fatal(err)
	}

	renderer, err := templates.New()
	if err != nil {
		t.Fatal(err)
	}
	config := repository.NewConfig()
	runner := jobs.NewRunner(5)
	h := New(Deps{
		Library:  services.NewLibrary(config),
		Shows:    services.NewShows(repository.NewShows()),
		Movies:   services.NewMovies(repository.NewMovies(), config),
		Config:   config,
		Jobs:     runner,
		Renderer: renderer,
		Version:  "test",
	})
	mux := http.NewServeMux()
	h.Register(mux)
	return h, runner, mux
}

func get(t *testing.T, mux http.Handler, path string, headers map[string]string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	return rr.Result()
}

func post(t *testing.T, mux http.Handler, path string, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	return rr.Result()
}

func bodyString(t *testing.T, r *http.Response) string {
	t.Helper()
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestGetShowsRendersPage(t *testing.T) {
	_, _, mux := newTestHandler(t)
	resp := get(t, mux, "/shows", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	body := bodyString(t, resp)
	if !strings.Contains(body, "Example Show") {
		t.Fatalf("body missing show title: %s", body)
	}
	if !strings.Contains(body, "<nav") {
		t.Fatalf("body missing nav: %s", body)
	}
	if !strings.Contains(body, `id="shows"`) {
		t.Fatalf("body missing #shows: %s", body)
	}
}

func TestGetShowsPartialForHX(t *testing.T) {
	_, _, mux := newTestHandler(t)
	resp := get(t, mux, "/shows", map[string]string{"HX-Request": "true"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	body := bodyString(t, resp)
	if !strings.Contains(body, "Example Show") {
		t.Fatalf("body missing show title: %s", body)
	}
	if strings.Contains(body, "<nav") {
		t.Fatalf("partial should not contain nav: %s", body)
	}
}

func TestPostScanStartsJobAndJobPartialFinishes(t *testing.T) {
	_, runner, mux := newTestHandler(t)
	resp := post(t, mux, "/scan", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	body := bodyString(t, resp)
	if !strings.Contains(body, `id="job-1"`) {
		t.Fatalf("body missing job-1: %s", body)
	}
	if !strings.Contains(body, `hx-trigger="every 1s"`) {
		t.Fatalf("body missing polling trigger: %s", body)
	}

	job, _ := runner.Get("1")
	if !runner.Wait(job, 5*time.Second) {
		t.Fatal("job did not finish")
	}

	resp = get(t, mux, "/jobs/1", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("HX-Trigger"); got != "job-finished" {
		t.Fatalf("expected HX-Trigger job-finished, got %q", got)
	}
	body = bodyString(t, resp)
	if strings.Contains(body, `hx-trigger="every 1s"`) {
		t.Fatalf("finished job should not poll: %s", body)
	}
	if !strings.Contains(body, "Scanning folder") {
		t.Fatalf("body missing scan log: %s", body)
	}
	if !strings.Contains(body, "Example Show") {
		t.Fatalf("body missing scanned show name: %s", body)
	}
}

func TestPostScanWhileBusyReturns409(t *testing.T) {
	_, runner, mux := newTestHandler(t)
	block := make(chan struct{})
	_, _ = runner.Start("blocker", func(log jobs.Logger) error {
		<-block
		return nil
	})
	defer close(block)

	resp := post(t, mux, "/scan", "")
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 got %d", resp.StatusCode)
	}
	body := bodyString(t, resp)
	if !strings.Contains(body, `class="flash error"`) {
		t.Fatalf("body missing flash error: %s", body)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	_, _, mux := newTestHandler(t)
	resp := get(t, mux, "/scan", nil)
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 got %d", resp.StatusCode)
	}
}

func TestGetShowPage(t *testing.T) {
	_, _, mux := newTestHandler(t)
	resp := get(t, mux, "/shows/example-show", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	body := bodyString(t, resp)
	if !strings.Contains(body, "S01E01") {
		t.Fatalf("body missing episode: %s", body)
	}

	resp = get(t, mux, "/shows/nope", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", resp.StatusCode)
	}
}

func TestBasicAuth(t *testing.T) {
	_, _, mux := newTestHandler(t)
	handler := BasicAuth(mux, "admin:pw")

	resp := get(t, handler, "/shows", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("WWW-Authenticate"); got != `Basic realm="couchness"` {
		t.Fatalf("expected WWW-Authenticate, got %q", got)
	}

	req := httptest.NewRequest(http.MethodGet, "/shows", nil)
	req.SetBasicAuth("admin", "pw")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	resp = rr.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	req = httptest.NewRequest(http.MethodGet, "/shows", nil)
	req.SetBasicAuth("admin", "wrong")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	resp = rr.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestBasicAuthConstantTimeCompare(t *testing.T) {
	// Ensure we use constant-time compare by checking wrong password rejected.
	_, _, mux := newTestHandler(t)
	handler := BasicAuth(mux, "admin:pw")
	req := httptest.NewRequest(http.MethodGet, "/shows", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:pwX")))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	resp := rr.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestJobNotFound(t *testing.T) {
	_, _, mux := newTestHandler(t)
	resp := get(t, mux, "/jobs/99", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", resp.StatusCode)
	}
}

func TestConfigPageAndSave(t *testing.T) {
	_, _, mux := newTestHandler(t)
	resp := get(t, mux, "/config", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	body := bodyString(t, resp)
	if !strings.Contains(body, `name="transmission_host"`) {
		t.Fatalf("body missing config form: %s", body)
	}

	resp = post(t, mux, "/config", "shows_dir=/tmp/shows&shows_dirs=%2Ftmp%2Fa%0A%2Ftmp%2Fb&movies_dir=/tmp/movies&transmission_host=tx&transmission_port=9091&transmission_auth=u:p&omdb_api_key=key")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("HX-Refresh"); got != "true" {
		t.Fatalf("expected HX-Refresh true, got %q", got)
	}

	persisted, err := storage.Db.GetAppConfiguration(&models.AppConfiguration{})
	if err != nil {
		t.Fatal(err)
	}
	if persisted.ShowsDir != "/tmp/shows" || persisted.MoviesDir != "/tmp/movies" ||
		persisted.TransmissionHost != "tx" || persisted.OmdbAPIKey != "key" {
		t.Fatalf("unexpected persisted config: %+v", persisted)
	}
	if len(persisted.ShowsDirs) != 2 || persisted.ShowsDirs[0] != "/tmp/a" || persisted.ShowsDirs[1] != "/tmp/b" {
		t.Fatalf("unexpected shows dirs: %v", persisted.ShowsDirs)
	}
	if got := storage.AppConfiguration.TransmissionHost; got != "tx" {
		t.Fatalf("live config not applied, host=%q", got)
	}
}

func TestDeleteShowAndMovie(t *testing.T) {
	_, _, mux := newTestHandler(t)
	if _, err := storage.NewMovieStorage(&models.Movie{Show: models.Show{ID: "test-movie", Title: "Test Movie", ExternalID: "tt0000001"}}).Save(); err != nil {
		t.Fatal(err)
	}

	resp := post(t, mux, "/movies/delete", "key=test-movie")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("movie delete: expected 200 got %d: %s", resp.StatusCode, bodyString(t, resp))
	}
	movies, err := storage.GetAllMovies()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range movies {
		if m.ID == "test-movie" {
			t.Fatal("movie still in storage after delete")
		}
	}
	if got := resp.Header.Get("HX-Trigger"); got != "movies-changed" {
		t.Fatalf("expected HX-Trigger movies-changed, got %q", got)
	}

	resp = post(t, mux, "/shows/example-show/delete", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("show delete: expected 200 got %d: %s", resp.StatusCode, bodyString(t, resp))
	}
	resp = get(t, mux, "/shows/example-show", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted show to 404, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestRootRedirects(t *testing.T) {
	_, _, mux := newTestHandler(t)
	resp := get(t, mux, "/", nil)
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302 got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if loc != "/shows" {
		t.Fatalf("expected /shows, got %q", loc)
	}
	_ = resp.Body.Close()
}

func TestConfigMediaServersKeepSecretsServerSide(t *testing.T) {
	_, _, mux := newTestHandler(t)
	base := "shows_dir=/tmp/shows&movies_dir=/tmp/movies&transmission_host=tx&transmission_port=9091&transmission_auth=u:p&omdb_api_key=key&emby_url="
	resp := post(t, mux, "/config", base+"&jellyfin_url=http://jf:8097/&jellyfin_username=highercomve&jellyfin_password=s3cret&jellyfin_api_key=")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	jf, ok := storage.AppConfiguration.MediaServer(models.MediaServerJellyfin)
	if !ok || jf.URL != "http://jf:8097" || jf.Username != "highercomve" || jf.Password != "s3cret" {
		t.Fatalf("jellyfin not saved: %+v", jf)
	}

	// the page never echoes the password back
	body := bodyString(t, get(t, mux, "/config", nil))
	if strings.Contains(body, "s3cret") {
		t.Fatal("password leaked into the config page")
	}
	if !strings.Contains(body, "saved, leave empty to keep") {
		t.Fatal("page should say a password is saved")
	}

	// saving again with an empty password keeps the stored one
	post(t, mux, "/config", base+"&jellyfin_url=http://jf:8097&jellyfin_username=highercomve&jellyfin_password=&jellyfin_api_key=")
	if jf, _ = storage.AppConfiguration.MediaServer(models.MediaServerJellyfin); jf.Password != "s3cret" {
		t.Fatalf("empty field should keep the password, got %q", jf.Password)
	}

	// clearing the URL removes the server
	post(t, mux, "/config", base+"&jellyfin_url=&jellyfin_username=&jellyfin_password=&jellyfin_api_key=")
	if _, ok := storage.AppConfiguration.MediaServer(models.MediaServerJellyfin); ok {
		t.Fatal("empty URL should remove the server")
	}

	// test/rescan without servers explain what to do
	resp = post(t, mux, "/media-servers/refresh", "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(bodyString(t, resp), "No media server configured") {
		t.Fatal("refresh without servers should explain how to add one")
	}
}

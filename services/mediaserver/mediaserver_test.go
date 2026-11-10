package mediaserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/highercomve/couchness/models"
)

// fakeServer imitates the Jellyfin/Emby endpoints couchness uses. prefix is
// "" for Jellyfin and "/emby" for Emby.
func fakeServer(t *testing.T, prefix string) (*httptest.Server, *[]string) {
	var mu sync.Mutex
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls = append(calls, r.Method+" "+r.URL.Path)
		mu.Unlock()
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "MediaBrowser ") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch r.Method + " " + r.URL.Path {
		case "POST " + prefix + "/Users/AuthenticateByName":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["Username"] != "highercomve" || body["Pw"] != "secret" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"AccessToken":"tok123","User":{"Name":"highercomve"}}`))
		case "POST " + prefix + "/Library/Refresh":
			if !strings.Contains(auth, `Token="tok123"`) && !strings.Contains(auth, `Token="apikey"`) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case "GET " + prefix + "/System/Info":
			if !strings.Contains(auth, "Token=") {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"ServerName":"rockpi-4b","Version":"10.11.11"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func TestRefreshWithUsernameAndPassword(t *testing.T) {
	srv, calls := fakeServer(t, "")
	c := New(models.MediaServer{Type: models.MediaServerJellyfin, URL: srv.URL, Username: "highercomve", Password: "secret"})
	if err := c.RefreshLibraries(); err != nil {
		t.Fatal(err)
	}
	want := []string{"POST /Users/AuthenticateByName", "POST /Library/Refresh"}
	if strings.Join(*calls, ",") != strings.Join(want, ",") {
		t.Fatalf("calls = %v, want %v", *calls, want)
	}
	info, err := c.Info()
	if err != nil || info.ServerName != "rockpi-4b" {
		t.Fatalf("info = %+v, %v", info, err)
	}
	if len(*calls) != 3 {
		t.Errorf("token should be reused, calls = %v", *calls)
	}
}

func TestRefreshWithAPIKeySkipsSignIn(t *testing.T) {
	srv, calls := fakeServer(t, "")
	c := New(models.MediaServer{Type: models.MediaServerJellyfin, URL: srv.URL, APIKey: "apikey", Username: "ignored"})
	if err := c.RefreshLibraries(); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 1 || (*calls)[0] != "POST /Library/Refresh" {
		t.Fatalf("calls = %v", *calls)
	}
}

func TestEmbyUsesItsPrefix(t *testing.T) {
	srv, calls := fakeServer(t, "/emby")
	c := New(models.MediaServer{Type: models.MediaServerEmby, URL: srv.URL + "/", Username: "highercomve", Password: "secret"})
	if err := c.RefreshLibraries(); err != nil {
		t.Fatal(err)
	}
	if (*calls)[0] != "POST /emby/Users/AuthenticateByName" {
		t.Fatalf("calls = %v", *calls)
	}
}

func TestWrongPasswordIsReported(t *testing.T) {
	srv, _ := fakeServer(t, "")
	c := New(models.MediaServer{Type: models.MediaServerJellyfin, URL: srv.URL, Username: "highercomve", Password: "nope"})
	err := c.RefreshLibraries()
	if err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("want an unauthorized error, got %v", err)
	}
}

func TestRefreshAllSkipsUnconfigured(t *testing.T) {
	srv, _ := fakeServer(t, "")
	results := RefreshAll([]models.MediaServer{
		{Type: models.MediaServerJellyfin, URL: srv.URL, Username: "highercomve", Password: "secret"},
		{Type: models.MediaServerEmby}, // no URL: ignored
	})
	if len(results) != 1 || results[0].Err != nil {
		t.Fatalf("results = %+v", results)
	}
}

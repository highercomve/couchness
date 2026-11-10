package common

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTorrentPathsHandshakeAndAuth(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if u, p, ok := r.BasicAuth(); !ok || u != "transmission" || p != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("X-Transmission-Session-Id") != "abc" {
			w.Header().Set("X-Transmission-Session-Id", "abc")
			w.WriteHeader(http.StatusConflict)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"fields":["name","downloadDir"]`) {
			t.Errorf("unexpected request body %s", body)
		}
		// trackerStats with a numeric lastScrapeTimedOut used to break decoding
		_, _ = io.WriteString(w, `{"result":"success","arguments":{"torrents":[
			{"name":"Inside.Llewyn.Davis.2013.1080p.BluRay.DTS-HD.MA.5.1.x264-PublicHD","downloadDir":"/mnt/cloud/movies/","trackerStats":[{"lastScrapeTimedOut":0}]},
			{"name":"Some.Show.S01E01.mkv","downloadDir":"/mnt/media2/series/some-show"}]}}`)
	}))
	defer srv.Close()

	paths, err := torrentPaths(srv.URL, "transmission", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Errorf("expected handshake + request, got %d calls", calls)
	}
	want := "/mnt/cloud/movies/Inside.Llewyn.Davis.2013.1080p.BluRay.DTS-HD.MA.5.1.x264-PublicHD"
	if paths[want] == "" {
		t.Errorf("missing %s in %v", want, paths)
	}
	if len(paths) != 2 {
		t.Errorf("want 2 torrents, got %v", paths)
	}
}

func TestTorrentPathsReportsErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	if _, err := torrentPaths(srv.URL, "x", "y"); err == nil {
		t.Fatal("expected an error for HTTP 401")
	}
}

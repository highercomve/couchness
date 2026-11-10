package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/highercomve/couchness/storage"
)

// TorrentPaths returns, for every torrent transmission still holds, the path
// of its data (download dir + torrent name) mapped to the torrent name.
//
// It talks to the RPC directly and asks only for "name" and "downloadDir":
// the transmission client library decodes every torrent field and fails on
// newer daemons (e.g. trackerStats.lastScrapeTimedOut became a number).
func TorrentPaths() (map[string]string, error) {
	return torrentPaths(transmissionRPCURL(), transmissionUser(), transmissionPassword())
}

func transmissionRPCURL() string {
	return fmt.Sprintf("http://%s:%s/transmission/rpc",
		storage.AppConfiguration.TransmissionHost, storage.AppConfiguration.TransmissionPort)
}

func transmissionUser() string {
	return strings.SplitN(storage.AppConfiguration.TransmissionAuth, ":", 2)[0]
}

func transmissionPassword() string {
	parts := strings.SplitN(storage.AppConfiguration.TransmissionAuth, ":", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

func torrentPaths(rpcURL, user, password string) (map[string]string, error) {
	body := []byte(`{"method":"torrent-get","arguments":{"fields":["name","downloadDir"]}}`)
	client := &http.Client{Timeout: 15 * time.Second}
	session := ""
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequest(http.MethodPost, rpcURL, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if session != "" {
			req.Header.Set("X-Transmission-Session-Id", session)
		}
		if user != "" {
			req.SetBasicAuth(user, password)
		}
		res, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		raw, err := io.ReadAll(io.LimitReader(res.Body, 32<<20))
		res.Body.Close()
		if err != nil {
			return nil, err
		}
		switch res.StatusCode {
		case http.StatusConflict: // CSRF handshake: retry with the session id
			session = res.Header.Get("X-Transmission-Session-Id")
			continue
		case http.StatusOK:
		default:
			return nil, fmt.Errorf("transmission RPC: HTTP %d", res.StatusCode)
		}
		var out struct {
			Result    string `json:"result"`
			Arguments struct {
				Torrents []struct {
					Name        string `json:"name"`
					DownloadDir string `json:"downloadDir"`
				} `json:"torrents"`
			} `json:"arguments"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, fmt.Errorf("transmission RPC: %w", err)
		}
		if out.Result != "success" {
			return nil, fmt.Errorf("transmission RPC: %s", out.Result)
		}
		paths := make(map[string]string, len(out.Arguments.Torrents))
		for _, t := range out.Arguments.Torrents {
			if t.Name == "" || t.DownloadDir == "" {
				continue
			}
			paths[filepath.Clean(filepath.Join(t.DownloadDir, t.Name))] = t.Name
		}
		return paths, nil
	}
	return nil, fmt.Errorf("transmission RPC: session handshake failed")
}

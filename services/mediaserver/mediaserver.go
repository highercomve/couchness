// Package mediaserver talks to Jellyfin and Emby: it signs in with an API key
// or a username/password and asks the server to rescan its libraries.
package mediaserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/highercomve/couchness/models"
)

// Client is a signed-in session with one Jellyfin or Emby server.
type Client struct {
	server models.MediaServer
	http   *http.Client
	token  string
}

// Info is what the server reports about itself.
type Info struct {
	ServerName string `json:"ServerName"`
	Version    string `json:"Version"`
}

// New creates a client; nothing is sent until the first call.
func New(server models.MediaServer) *Client {
	return &Client{server: server, http: &http.Client{Timeout: 30 * time.Second}}
}

// Name is a label for messages, e.g. "Jellyfin (http://localhost:8097)".
func (c *Client) Name() string {
	kind := "Jellyfin"
	if c.server.Type == models.MediaServerEmby {
		kind = "Emby"
	}
	return fmt.Sprintf("%s (%s)", kind, c.server.URL)
}

// base returns the API root: Emby serves its API under /emby.
func (c *Client) base() string {
	u := strings.TrimRight(c.server.URL, "/")
	if c.server.Type == models.MediaServerEmby && !strings.HasSuffix(strings.ToLower(u), "/emby") {
		u += "/emby"
	}
	return u
}

func deviceID() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "host"
	}
	// stable per machine so the server lists one device, not one per run
	return "couchness-" + host
}

func (c *Client) authHeader() string {
	h := fmt.Sprintf(`MediaBrowser Client="couchness", Device="couchness", DeviceId="%s", Version="1.0"`, deviceID())
	if c.token != "" {
		h += fmt.Sprintf(`, Token="%s"`, c.token)
	}
	return h
}

// signIn obtains a token: the API key as-is, or AuthenticateByName.
func (c *Client) signIn() error {
	if c.token != "" {
		return nil
	}
	if c.server.APIKey != "" {
		c.token = c.server.APIKey
		return nil
	}
	if c.server.Username == "" {
		return errors.New("no API key or username configured")
	}
	body, _ := json.Marshal(map[string]string{"Username": c.server.Username, "Pw": c.server.Password})
	var out struct {
		AccessToken string `json:"AccessToken"`
	}
	if err := c.do(http.MethodPost, "/Users/AuthenticateByName", body, &out); err != nil {
		return fmt.Errorf("sign in as %q: %w", c.server.Username, err)
	}
	if out.AccessToken == "" {
		return errors.New("sign in: the server returned no access token")
	}
	c.token = out.AccessToken
	return nil
}

func (c *Client) do(method, path string, body []byte, out any) error {
	req, err := http.NewRequest(method, c.base()+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("X-Emby-Authorization", c.authHeader())
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	switch {
	case res.StatusCode == http.StatusUnauthorized:
		return errors.New("unauthorized: wrong username/password or API key")
	case res.StatusCode >= 300:
		msg := strings.TrimSpace(string(raw))
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return fmt.Errorf("HTTP %d %s", res.StatusCode, msg)
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("unexpected reply: %w", err)
		}
	}
	return nil
}

// Info signs in and returns the server name and version (connection test).
func (c *Client) Info() (*Info, error) {
	if err := c.signIn(); err != nil {
		return nil, err
	}
	info := &Info{}
	if err := c.do(http.MethodGet, "/System/Info", nil, info); err != nil {
		return nil, err
	}
	return info, nil
}

// RefreshLibraries asks the server to scan all its libraries. The scan runs
// in the background on the server; this returns once it has started.
func (c *Client) RefreshLibraries() error {
	if err := c.signIn(); err != nil {
		return err
	}
	return c.do(http.MethodPost, "/Library/Refresh", nil, nil)
}

// Result is the outcome of RefreshAll for one server.
type Result struct {
	Name string
	Err  error
}

// RefreshAll rescans every configured server and reports each outcome.
func RefreshAll(servers []models.MediaServer) []Result {
	var out []Result
	for _, s := range servers {
		if !s.Configured() {
			continue
		}
		c := New(s)
		out = append(out, Result{Name: c.Name(), Err: c.RefreshLibraries()})
	}
	return out
}

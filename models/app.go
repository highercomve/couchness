package models

// AppConfiguration couchness app configuration
type AppConfiguration struct {
	Name             string   `json:"name,omitempty"`
	ShowsDir         string   `json:"media_dir,omitempty"`
	ShowsDirs        []string `json:"media_directories,omitempty"`
	MoviesDir        string   `json:"movies_dir,omitempty"`
	OmdbAPIKey       string   `json:"omdb_api_key"`
	TransmissionHost string   `json:"transmission_host"`
	TransmissionPort string   `json:"transmission_port"`
	TransmissionAuth string   `json:"transmission_auth"`
	// MediaServers are Jellyfin/Emby servers couchness asks to rescan after
	// it changes the library (e.g. `couchness movies group --apply`).
	MediaServers []MediaServer `json:"media_servers,omitempty"`
}

// Media server types.
const (
	MediaServerJellyfin = "jellyfin"
	MediaServerEmby     = "emby"
)

// MediaServer is a Jellyfin or Emby server couchness can sign in to. Either
// an API key or a username/password pair is used; the API key wins.
type MediaServer struct {
	Type     string `json:"type"` // "jellyfin" or "emby"
	URL      string `json:"url"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	APIKey   string `json:"api_key,omitempty"`
}

// Configured reports whether the server has a URL and some credentials.
func (m MediaServer) Configured() bool {
	return m.URL != "" && (m.APIKey != "" || m.Username != "")
}

// MediaServer returns the configured server of the given type.
func (c *AppConfiguration) MediaServer(kind string) (MediaServer, bool) {
	for _, m := range c.MediaServers {
		if m.Type == kind {
			return m, true
		}
	}
	return MediaServer{Type: kind}, false
}

// SetMediaServer adds or replaces the server of m.Type; an empty URL removes it.
func (c *AppConfiguration) SetMediaServer(m MediaServer) {
	out := c.MediaServers[:0:0]
	for _, existing := range c.MediaServers {
		if existing.Type != m.Type {
			out = append(out, existing)
		}
	}
	if m.URL != "" {
		out = append(out, m)
	}
	c.MediaServers = out
}

package handlers

import (
	"net/http"
	"strings"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/services/mediaserver"
	viewmodels "github.com/highercomve/couchness/web/models"
)

var mediaServerKinds = []struct{ Type, Label string }{
	{models.MediaServerJellyfin, "Jellyfin"},
	{models.MediaServerEmby, "Emby"},
}

func (h *Handler) configPage(w http.ResponseWriter, r *http.Request) {
	data := viewmodels.ConfigPage{}
	for _, k := range mediaServerKinds {
		m, _ := h.Config.Get().MediaServer(k.Type)
		data.MediaServers = append(data.MediaServers, viewmodels.MediaServerForm{
			Type: k.Type, Label: k.Label, URL: m.URL, Username: m.Username,
			HasPassword: m.Password != "", HasAPIKey: m.APIKey != "",
		})
	}
	h.render(w, http.StatusOK, "config", h.page("Configuration", "config", data))
}

func (h *Handler) configSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.fail(w, http.StatusBadRequest, err)
		return
	}
	c := *h.Config.Get()
	c.ShowsDir = strings.TrimSpace(r.FormValue("shows_dir"))
	c.ShowsDirs = splitLines(r.FormValue("shows_dirs"))
	c.MoviesDir = strings.TrimSpace(r.FormValue("movies_dir"))
	c.TransmissionHost = strings.TrimSpace(r.FormValue("transmission_host"))
	c.TransmissionPort = strings.TrimSpace(r.FormValue("transmission_port"))
	c.TransmissionAuth = strings.TrimSpace(r.FormValue("transmission_auth"))
	c.OmdbAPIKey = strings.TrimSpace(r.FormValue("omdb_api_key"))
	c.MediaServers = append([]models.MediaServer(nil), c.MediaServers...)
	for _, k := range mediaServerKinds {
		if _, present := r.Form[k.Type+"_url"]; !present {
			continue // form without media server fields: keep what is stored
		}
		existing, _ := c.MediaServer(k.Type)
		m := models.MediaServer{
			Type:     k.Type,
			URL:      strings.TrimRight(strings.TrimSpace(r.FormValue(k.Type+"_url")), "/"),
			Username: strings.TrimSpace(r.FormValue(k.Type + "_username")),
			Password: r.FormValue(k.Type + "_password"),
			APIKey:   strings.TrimSpace(r.FormValue(k.Type + "_api_key")),
		}
		// secrets are never sent to the browser: an empty field keeps the saved one
		if m.Password == "" && r.FormValue(k.Type+"_clear_password") == "" && m.Username == existing.Username {
			m.Password = existing.Password
		}
		if m.APIKey == "" && r.FormValue(k.Type+"_clear_api_key") == "" {
			m.APIKey = existing.APIKey
		}
		c.SetMediaServer(m)
	}
	if err := h.Config.Save(&c); err != nil {
		h.fail(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("HX-Refresh", "true")
	h.flash(w, http.StatusOK, "ok", "Configuration saved")
}

// mediaServersTest signs in to every configured server.
func (h *Handler) mediaServersTest(w http.ResponseWriter, r *http.Request) {
	var lines []string
	failed := false
	for _, s := range h.Config.Get().MediaServers {
		if !s.Configured() {
			continue
		}
		client := mediaserver.New(s)
		info, err := client.Info()
		if err != nil {
			failed = true
			lines = append(lines, client.Name()+": "+err.Error())
			continue
		}
		lines = append(lines, client.Name()+": signed in to "+info.ServerName+" (version "+info.Version+")")
	}
	h.mediaServerFlash(w, lines, failed)
}

// mediaServersRefresh asks every configured server to rescan its libraries.
func (h *Handler) mediaServersRefresh(w http.ResponseWriter, r *http.Request) {
	var lines []string
	failed := false
	for _, res := range mediaserver.RefreshAll(h.Config.Get().MediaServers) {
		if res.Err != nil {
			failed = true
			lines = append(lines, res.Name+": rescan failed: "+res.Err.Error())
			continue
		}
		lines = append(lines, res.Name+": library rescan started")
	}
	h.mediaServerFlash(w, lines, failed)
}

func (h *Handler) mediaServerFlash(w http.ResponseWriter, lines []string, failed bool) {
	if len(lines) == 0 {
		h.flash(w, http.StatusOK, "error", "No media server configured: add a Jellyfin or Emby URL and credentials, then save.")
		return
	}
	// htmx only swaps 2xx replies, so failures are 200 with an error style
	kind := "ok"
	if failed {
		kind = "error"
	}
	h.flash(w, http.StatusOK, kind, strings.Join(lines, " · "))
}

func splitLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}

// Package models holds the view models rendered by the web templates.
package models

import (
	"github.com/highercomve/couchness/common"
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/web/jobs"
)

// Page wraps every full page render.
type Page struct {
	Title      string
	Active     string
	Version    string
	Config     *models.AppConfiguration
	RunningJob *jobs.Snapshot
	Data       interface{}
}

// Flash is a short status message.
type Flash struct {
	Kind    string // "ok" or "error"
	Message string
}

// ShowsPage lists the tracked shows.
type ShowsPage struct {
	Shows []*models.Show
	Dirs  []string
}

// ShowPage shows one show with its episodes and the IMDb identify widget.
type ShowPage struct {
	Show    *models.Show
	Query   string
	Results []common.OmdbResults
	Error   string
}

// MoviesPage lists downloaded movies.
type MoviesPage struct {
	Movies    []*models.Movie
	MoviesDir string
	Error     string
}

// MovieSearch holds OMDb search results for movies.
type MovieSearch struct {
	Query   string
	Results []common.OmdbResults
	Error   string
}

// MovieTorrents holds torrent versions found for a movie.
type MovieTorrents struct {
	Movie    *models.Movie
	Torrents models.Episodes
	Error    string
}

// MovieRequest are the fields accepted to identify a movie to download.
type MovieRequest struct {
	ImdbID string
	Title  string
	Year   string
	Key    string
	Folder string
}

// ShowConfigRequest are the fields accepted to edit a show configuration.
type ShowConfigRequest struct {
	FollowType string
	Since      int
	Resolution string
	Quality    string
	Codec      string
	Services   []string
}

// TorrentRequest are the fields accepted to queue a torrent.
type TorrentRequest struct {
	Title      string
	Magnet     string
	Size       int64
	Seeds      int
	Resolution string
	Quality    string
	Codec      string
}

// ConfigPage is the configuration page's extra data.
type ConfigPage struct {
	MediaServers []MediaServerForm
}

// MediaServerForm shows one media server's settings without its secrets.
type MediaServerForm struct {
	Type        string // "jellyfin" or "emby"
	Label       string // "Jellyfin"
	URL         string
	Username    string
	HasPassword bool
	HasAPIKey   bool
}

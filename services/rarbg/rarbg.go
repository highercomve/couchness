package rarbg

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/utils"
	"github.com/qopher/go-torrentapi"
)

const (
	//ServiceType showrss service type
	ServiceType = "rarbg"
)

// Service show rss service
type Service struct {
	ID      string `json:"id"`
	BaseURL string `json:"base_url"`
}

// GetID get service ID
func (s Service) GetID() string {
	return s.ID
}

// GetURL get service base URL
func (s Service) GetURL() string {
	return s.BaseURL
}

// ShowURL get show information URL
func (s Service) ShowURL(showID string, page, limit int) string {
	return fmt.Sprintf("%s?imdb_id=%s&page=%d&limit=%d", s.BaseURL, showID, page, limit)
}

// GetShowData get show data from service
func (s Service) GetShowData(show *models.Show, page, limit int) (*models.Show, error) {
	if show.ExternalID == "" {
		return nil, errors.New("Show " + show.ID + " doesn't have a external ID")
	}

	api, err := torrentapi.New("cli")
	api.
		Category(2).
		Category(18).
		Category(41).
		Category(49).
		Format("json_extended").
		Limit(limit).
		SearchIMDb(show.ExternalID)

	results, err := api.Search()
	if err != nil {
		fmt.Printf("Error while querying torrentapi %s", err)
		return nil, err
	}

	show.TorrentCount = len(results)
	for _, torrent := range results {
		torrentInfo, err := utils.ParseTorrent(torrent.Title)
		if err != nil {
			continue
		}
		torrentInfo.MagnetURL = torrent.Download
		torrentInfo.Downloaded = false
		torrentInfo.Extension = filepath.Ext(torrent.Title)
		torrentInfo.Location = filepath.Join(show.Directory, torrent.Title)
		torrentInfo.Seeds = torrent.Seeders
		torrentInfo.Size = int64(torrent.Size)
		show.Episodes = append(show.Episodes, torrentInfo)
	}

	return show, nil
}

// New create new show rss follow service
func New() Service {
	return Service{
		ID:      string(ServiceType),
		BaseURL: "https://torrentapi.org/pubapi_v2.php",
	}
}

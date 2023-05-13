package eztv

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/utils"
)

const (
	//ServiceType showrss service type
	ServiceType = "eztv"
)

// Service show rss service
type Service struct {
	ID      string `json:"id"`
	BaseURL string `json:"base_url"`
}

type eztvResponse struct {
	Imdb         string `json:"imdb_id"`
	TorrentCount int    `json:"torrents_count"`
	Limit        int    `json:"limit"`
	Page         int    `json:"page"`
	Torrents     []torrent
}

type torrent struct {
	ID               int    `json:"id"`
	Hash             string `json:"hash"`
	Filename         string `json:"filename"`
	EpisodeURL       string `json:"episode_url"`
	TorrentURL       string `json:"torrent_url"`
	MagnetURL        string `json:"magnet_url"`
	Title            string `json:"title"`
	ImdbID           string `json:"imdb_id"`
	Season           string `json:"season"`
	Episode          string `json:"episode"`
	Seeds            int    `json:"seeds"`
	Peers            int    `json:"peers"`
	DateReleasedUnix int64  `json:"date_released_unix"`
	Size             string `json:"size_bytes"`
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
func (s Service) GetShowData(show *models.Show, page, limit int, typeOf string) (*models.Show, error) {
	if show.ExternalID == "" {
		return nil, errors.New("Show " + show.ID + " doesn't have a external ID")
	}

	imdbID := strings.ReplaceAll(show.ExternalID, "tt", "")
	url := s.ShowURL(imdbID, page, limit)
	resp, err := http.Get(url)
	if err != nil {
		return show, err
	}
	response, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return show, err
	}

	eztvShow := &eztvResponse{}
	err = json.Unmarshal(response, eztvShow)
	if err != nil {
		return show, err
	}

	if eztvShow == nil {
		return nil, errors.New("Imbb show ID as an error " + show.ExternalID)
	}
	show.TorrentCount = eztvShow.TorrentCount
	for _, torrent := range eztvShow.Torrents {
		torrentInfo, err := utils.ParseTorrent(torrent.Filename)
		if err != nil {
			continue
		}
		torrentInfo.Title = torrent.Filename
		torrentInfo.MagnetURL = torrent.MagnetURL
		torrentInfo.Downloaded = false
		torrentInfo.Extension = filepath.Ext(torrent.Filename)
		torrentInfo.Location = filepath.Join(show.Directory, torrent.Filename)
		torrentInfo.Seeds = torrent.Seeds
		size, err := strconv.ParseInt(torrent.Size, 10, 64)
		if err == nil {
			torrentInfo.Size = size
		}
		show.Episodes = append(show.Episodes, torrentInfo)
	}

	return show, nil
}

// New create new show rss follow service
func New() Service {
	return Service{
		ID:      string(ServiceType),
		BaseURL: "https://eztv.ag/api/get-torrents",
	}
}

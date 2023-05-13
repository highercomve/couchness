package models

import (
	"fmt"

	"github.com/dustin/go-humanize"
)

// TorrentInfo is the resulting structure returned by Parse
type TorrentInfo struct {
	Title      string `json:"-"`
	Name       string `json:"-"`
	Extension  string `json:"extension"`
	Location   string `json:"location"`
	MagnetURL  string `json:"magnet_url,omitempty"`
	Downloaded bool   `json:"downloaded,omitempty"`
	Seeds      int    `json:"seeds,omitempty"`
	Season     int    `json:"season,omitempty"`
	Episode    int    `json:"episode,omitempty"`
	Year       int    `json:"year,omitempty"`
	Resolution string `json:"resolution,omitempty"`
	Quality    string `json:"quality,omitempty"`
	Codec      string `json:"codec,omitempty"`
	Audio      string `json:"audio,omitempty"`
	Group      string `json:"group,omitempty"`
	Region     string `json:"region,omitempty"`
	Extended   bool   `json:"extended,omitempty"`
	Hardcoded  bool   `json:"hardcoded,omitempty"`
	Proper     bool   `json:"proper,omitempty"`
	Repack     bool   `json:"repack,omitempty"`
	Container  string `json:"container,omitempty"`
	Widescreen bool   `json:"widescreen,omitempty"`
	Website    string `json:"website,omitempty"`
	Language   string `json:"language,omitempty"`
	Sbs        string `json:"sbs,omitempty"`
	Unrated    bool   `json:"unrated,omitempty"`
	Size       int64  `json:"size,omitempty"`
	ThreeD     bool   `json:"3d,omitempty"`
}

func (torrent *TorrentInfo) Summary() string {
	return fmt.Sprintf(
		"%s \t Size: %s \t Seeds: %d",
		torrent.Title,
		humanize.Bytes(uint64(torrent.Size)),
		torrent.Seeds,
	)
}

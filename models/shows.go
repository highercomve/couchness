package models

import (
	"fmt"
)

const (
	// FollowTypeLatest follow lastest episode
	FollowTypeLatest = "latest"

	// FollowTypeSince follow since season N
	FollowTypeSince = "since"

	// FollowTypeAll follow all the episodes of the series
	FollowTypeAll = "all"
)

// Episodes an array of torrent information
type Episodes []*TorrentInfo

// Show how a show is defined
type Show struct {
	// sync.RWMutex
	ID            string `json:"id"`
	Title         string `json:"title"`
	ExternalID    string `json:"external-id"`
	Directory     string `json:"directory"`
	TorrentCount  int    `json:"torrents_count"`
	Configuration *ShowConf
	Episodes      Episodes
}

// ShowConf show configuration
type ShowConf struct {
	FollowType string `json:"follow_type"`
	Service    string `json:"service"`
	Since      int    `json:"since"`
	Quality    string `json:"quality"`
	Codec      string `json:"codec"`
	Resolution string `json:"resolution"`
	FilterBy   string `json:"filter-by"`
}

// ShowsMap show torrent information
type ShowsMap map[string]*Show

// PrintString covert show data into string
func (s *Show) PrintString() string {
	return fmt.Sprintf("%s - external ID %s - internal ID %s", s.Title, s.ExternalID, s.ID)
}

// // AddOrUpdateEpisode Add or update an episode if already exist
// func (s *Show) AddOrUpdateEpisode(*TorrentInfo) (*Show, error) {
// 	return s, nil
// }

// // Save save on database
// func (s *Show) Save() (*Show, error) {
// 	s.Lock()
// 	defer s.Unlock()

// 	err := storage.Db.Driver.Write(storage.Db.Collections.Shows, s.ID, s)
// 	return s, err
// }

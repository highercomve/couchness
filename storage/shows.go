package storage

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/highercomve/couchness/models"
)

// ShowStorage storage methods for shows
type ShowStorage struct {
	*models.Show
}

// NewShowStorage create new ShowStorage
func NewShowStorage(show *models.Show) *ShowStorage {
	return &ShowStorage{
		show,
	}
}

// AddOrUpdateEpisode Add or update an episode if already exist
func (s *ShowStorage) AddOrUpdateEpisode(*models.TorrentInfo) (*ShowStorage, error) {
	return s, nil
}

// Save save on database
func (s *ShowStorage) Save() (*ShowStorage, error) {
	sort.SliceStable(s.Show.Episodes, func(i, j int) bool {
		se1 := (s.Show.Episodes[i].Season * 100) + s.Show.Episodes[i].Episode
		se2 := (s.Show.Episodes[j].Season * 100) + s.Show.Episodes[j].Episode

		return se1 > se2
	})

	err := Db.Driver.Write(Db.Collections.Shows, s.ID, s)
	return s, err
}

// SortEpisodes sort array of episodes
func SortEpisodes(episodes models.Episodes) {
	sort.SliceStable(episodes, func(i, j int) bool {
		se1 := (episodes[i].Season * 100) + episodes[i].Episode
		se2 := (episodes[j].Season * 100) + episodes[j].Episode

		return se1 > se2
	})
}

// GetAllShows get all shows
func GetAllShows() ([]*models.Show, error) {
	records, err := Db.Driver.ReadAll(Db.Collections.Shows)
	if err != nil {
		return nil, err
	}
	shows := []*models.Show{}
	for _, f := range records {
		showFound := &models.Show{}
		if err := json.Unmarshal([]byte(f), showFound); err != nil {
			fmt.Println("Error", err)
		}
		shows = append(shows, showFound)
	}

	return shows, nil
}

package storage

import "github.com/highercomve/couchness/models"

import "sort"

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

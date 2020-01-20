package common

import (
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

// GetShow get all shows if showID is empty and get one show if not
func GetShow(showID string, includeEpisodes bool) (interface{}, error) {
	show := &models.Show{}
	err := storage.Db.Driver.Read(storage.Db.Collections.Shows, showID, show)
	if err != nil {
		return nil, err
	}

	if !includeEpisodes {
		countEpisodes(show)
	}

	return show, nil
}

// GetShows get all shows
func GetShows() ([]*models.Show, error) {
	shows, err := storage.GetAllShows()
	if err != nil {
		return nil, err
	}
	for _, s := range shows {
		countEpisodes(s)
	}

	return shows, nil
}

func countEpisodes(s *models.Show) {
	s.EpisodesCount = len(s.Episodes)
	s.Episodes = make(models.Episodes, 0)
}

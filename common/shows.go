package common

import (
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

// GetShow get all shows if showID is empty and get one show if not
func GetShow(showID string) (interface{}, error) {
	if showID == "" {
		shows, err := storage.GetAllShows()
		if err != nil {
			return nil, err
		}
		for _, s := range shows {
			countEpisodes(s)
			shows = append(shows, s)
		}

		return shows, nil
	}

	show := &models.Show{}
	err := storage.Db.Driver.Read(storage.Db.Collections.Shows, showID, show)
	if err != nil {
		return nil, err
	}

	return show, nil
}

func countEpisodes(s *models.Show) {
	s.EpisodesCount = len(s.Episodes)
	s.Episodes = make(models.Episodes, 0)
}

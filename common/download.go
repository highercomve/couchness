package common

import (
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

// Download download show using the show id
func Download(showID string) error {
	show := &models.Show{}
	err := storage.Db.Driver.Read(storage.Db.Collections.Shows, showID, show)
	if err != nil {
		return err
	}

	_, err = Scan(show.Directory, show.ID, false, false)
	if err != nil {
		return err
	}

	return DownloadShow(show)
}

// DownloadEpisode download show using the show id and episode
func DownloadEpisode(showID, episode string, limit int) error {
	show := &models.Show{}
	err := storage.Db.Driver.Read(storage.Db.Collections.Shows, showID, show)
	if err != nil {
		return err
	}

	_, err = DownloadEpisodeOfShow(show, episode, limit)
	return err
}

package common

import (
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

// Update download show using the show id
func Update(showID string) error {
	show := &models.Show{}
	err := storage.Db.Driver.Read(storage.Db.Collections.Shows, showID, show)
	if err != nil {
		return err
	}

	show, err = ScanShowDir(show.Directory, show)
	if err != nil {
		return err
	}

	if err = DownloadShow(show); err != nil {
		return err
	}

	_, err = storage.NewShowStorage(show).Save()
	return err
}

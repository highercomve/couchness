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

	return DownloadShow(show)
}

package common

import (
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

// Disable disableShow
func DisableShow(showID string) (interface{}, error) {
	show := &models.Show{}
	err := storage.Db.Driver.Read(storage.Db.Collections.Shows, showID, show)
	if err != nil {
		return nil, err
	}

	show.Configuration.FollowType = models.FollowTypeManual

	err = storage.Db.Driver.Write(storage.Db.Collections.Shows, showID, show)
	if err != nil {
		return nil, err
	}

	return show, nil
}

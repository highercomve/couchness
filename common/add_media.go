package common

import (
	"github.com/highercomve/couchness/storage"
)

// AddMedia add new media directory to database
func AddMedia(mediaDirectory string) error {
	return storage.Db.AddMediaDir(mediaDirectory)
}

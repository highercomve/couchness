package storage

import (
	"os/user"

	scribble "github.com/sdomino/scribble"
)

// Storage manage storage
type Storage struct {
	Driver      *scribble.Driver
	Collections *Collections
}

// Collections All collections available
type Collections struct {
	Shows         string
	Configuration string
}

// Db database session
var Db = &Storage{}

// DbDir application configuration directory
var DbDir = ""

// Init initialize global Db
func Init() error {
	var err error

	usr, err := user.Current()
	if err != nil {
		return err
	}

	DbDir = usr.HomeDir + "/.couchness"
	Db, err = New(DbDir, nil)
	if err != nil {
		return err
	}

	AppConfiguration, err = Db.GetAppConfiguration()
	return err
}

// New create a new Storage
func New(dir string, options *scribble.Options) (*Storage, error) {
	db, err := scribble.New(dir, options)
	if err != nil {
		return nil, err
	}

	return &Storage{
		Driver: db,
		Collections: &Collections{
			Shows:         "shows",
			Configuration: "configuration",
		},
	}, nil
}

package storage

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/highercomve/couchness/models"
	scribble "github.com/sdomino/scribble"
)

// Storage manage storage
type Storage struct {
	Directory   string
	Driver      *scribble.Driver
	Collections *Collections
}

// Collections All collections available
type Collections struct {
	Movies        string
	Shows         string
	Configuration string
}

// Db database session
var Db = &Storage{}

// DbDir application configuration directory
var DbDir = ""

// Init initialize global Db
func Init(configDir string) error {
	var err error

	if configDir == "" {
		usr, err := user.Current()
		if err != nil {
			return err
		}
		configDir = usr.HomeDir + "/.couchness"
	} else {
		configDir, err = filepath.Abs(configDir)
		if err != nil {
			return err
		}
	}
	Db, err = New(configDir, nil)
	if err != nil {
		return err
	}

	AppConfiguration, err = Db.GetAppConfiguration(&models.AppConfiguration{
		MoviesDir:        os.Getenv("COUCHNESS_MOVIES_DIR"),
		ShowsDir:         os.Getenv("COUCHNESS_SHOWS_DIR"),
		ShowsDirs:        []string{os.Getenv("COUCHNESS_SHOWS_DIR")},
		OmdbAPIKey:       os.Getenv("COUCHNESS_OMDB_API_KEY"),
		TransmissionAuth: os.Getenv("COUCHNESS_TRANSMISSION_AUTH"),
		TransmissionHost: os.Getenv("COUCHNESS_TRANSMISSION_HOST"),
		TransmissionPort: os.Getenv("COUCHNESS_TRANSMISSION_PORT"),
	})
	return err
}

// New create a new Storage
func New(dir string, options *scribble.Options) (*Storage, error) {
	db, err := scribble.New(dir, options)
	if err != nil {
		return nil, err
	}

	return &Storage{
		Directory: dir,
		Driver:    db,
		Collections: &Collections{
			Movies:        "movies",
			Shows:         "shows",
			Configuration: "configuration",
		},
	}, nil
}

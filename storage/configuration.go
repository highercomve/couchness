package storage

import (
	"errors"
	"os"
	"os/user"

	"github.com/highercomve/couchness/models"
)

const (
	appConfID = "couchness"
)

// AppConfiguration app configuration global
var AppConfiguration = &models.AppConfiguration{}

// GetAppConfiguration get couchness configuration
func (s *Storage) GetAppConfiguration() (*models.AppConfiguration, error) {
	configuration := &models.AppConfiguration{}
	err := s.Driver.Read(s.Collections.Configuration, appConfID, configuration)
	if err == nil {
		return configuration, nil
	}

	usr, err := user.Current()
	if err != nil {
		return nil, errors.New("Can load os username")
	}

	configuration.MediaDir = os.Getenv("COUCHNESS_MEDIA_DIR")
	configuration.OmdbAPIKey = os.Getenv("COUCHNESS_OMDB_API_KEY")
	configuration.TransmissionAuth = os.Getenv("COUCHNESS_TRANSMISSION_AUTH")
	configuration.TransmissionHost = os.Getenv("COUCHNESS_TRANSMISSION_HOST")
	configuration.TransmissionPort = os.Getenv("COUCHNESS_TRANSMISSION_PORT")

	if configuration.MediaDir == "" {
		mediaDir := usr.HomeDir + "/couchnessMedia"
		err = os.Mkdir(mediaDir, os.FileMode(666))
		if err != nil {
			return nil, errors.New("Can create media folder: " + mediaDir)
		}

		configuration.MediaDir = mediaDir
	}

	if configuration.TransmissionAuth == "" {
		configuration.TransmissionAuth = "transmission:transmission"
	}

	if configuration.TransmissionHost == "" {
		configuration.TransmissionHost = "localhost"
	}

	if configuration.TransmissionPort == "" {
		configuration.TransmissionPort = "9091"
	}

	err = s.Driver.Write(s.Collections.Configuration, appConfID, configuration)

	return configuration, err
}

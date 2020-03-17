package storage

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"

	"github.com/highercomve/couchness/models"
	"github.com/swaggo/cli"
)

const (
	appConfID = "couchness"
)

// AppConfiguration app configuration global
var AppConfiguration = &models.AppConfiguration{}

// GetAppConfiguration get couchness configuration
func (s *Storage) GetAppConfiguration(configuration *models.AppConfiguration) (*models.AppConfiguration, error) {
	err := s.Driver.Read(s.Collections.Configuration, appConfID, configuration)
	if err == nil {
		return configuration, nil
	}

	if configuration.MediaDir == "" {
		usr, err := user.Current()
		if err != nil {
			return nil, errors.New("Can load os username")
		}
		mediaDir := usr.HomeDir + "/couchnessMedia"
		err = os.Mkdir(mediaDir, os.FileMode(666))
		if err != nil {
			return nil, errors.New("Can't create media folder: " + mediaDir)
		}

		configuration.MediaDir = mediaDir
		configuration.MediaDirs = []string{mediaDir}
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

// AddMediaDir add a new media directory
func (s *Storage) AddMediaDir(directory string) error {
	err := s.Driver.Read(s.Collections.Configuration, appConfID, AppConfiguration)
	if err == nil {
		return err
	}

	folderPath, err := filepath.Abs(directory)
	if err != nil {
		return cli.NewExitError(err.Error(), 0)
	}
	AppConfiguration.MediaDirs = append(AppConfiguration.MediaDirs, folderPath)

	return s.Driver.Write(s.Collections.Configuration, appConfID, AppConfiguration)
}

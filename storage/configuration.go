package storage

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/highercomve/couchness/models"
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

	if configuration.ShowsDir == "" {
		usr, err := user.Current()
		if err != nil {
			return nil, errors.New("Can load os username")
		}
		mediaDir := usr.HomeDir + "/couchnessMedia"
		err = os.MkdirAll(mediaDir, 0755)
		if err != nil {
			return nil, errors.New("Can't create media folder: " + mediaDir)
		}

		configuration.ShowsDir = mediaDir
		configuration.ShowsDirs = []string{mediaDir}
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

// SaveAppConfiguration persist the app configuration
func (s *Storage) SaveAppConfiguration(configuration *models.AppConfiguration) error {
	return s.Driver.Write(s.Collections.Configuration, appConfID, configuration)
}

// applyEnvironmentOverrides applies non-empty runtime settings over persisted configuration.
// Secrets and deployment-specific endpoints can change without rewriting the database.
func applyEnvironmentOverrides(configuration *models.AppConfiguration) {
	if value := os.Getenv("COUCHNESS_MOVIES_DIR"); value != "" {
		configuration.MoviesDir = value
	}
	if value := os.Getenv("COUCHNESS_SHOWS_DIR"); value != "" {
		configuration.ShowsDir = value
		found := false
		for _, directory := range configuration.ShowsDirs {
			if directory == value {
				found = true
				break
			}
		}
		if !found {
			configuration.ShowsDirs = append(configuration.ShowsDirs, value)
		}
	}
	if value := os.Getenv("COUCHNESS_OMDB_API_KEY"); value != "" {
		configuration.OmdbAPIKey = value
	}
	if value := os.Getenv("COUCHNESS_TRANSMISSION_AUTH"); value != "" {
		configuration.TransmissionAuth = value
	}
	if value := os.Getenv("COUCHNESS_TRANSMISSION_HOST"); value != "" {
		configuration.TransmissionHost = value
	}
	if value := os.Getenv("COUCHNESS_TRANSMISSION_PORT"); value != "" {
		configuration.TransmissionPort = value
	}
	for _, kind := range []string{models.MediaServerJellyfin, models.MediaServerEmby} {
		applyMediaServerOverrides(configuration, kind)
	}
}

// applyMediaServerOverrides reads COUCHNESS_JELLYFIN_URL, _USERNAME,
// _PASSWORD and _API_KEY (or the COUCHNESS_EMBY_ equivalents).
func applyMediaServerOverrides(configuration *models.AppConfiguration, kind string) {
	prefix := "COUCHNESS_" + strings.ToUpper(kind) + "_"
	server, _ := configuration.MediaServer(kind)
	changed := false
	for suffix, field := range map[string]*string{
		"URL": &server.URL, "USERNAME": &server.Username, "PASSWORD": &server.Password, "API_KEY": &server.APIKey,
	} {
		if value := os.Getenv(prefix + suffix); value != "" {
			*field = value
			changed = true
		}
	}
	if changed {
		configuration.SetMediaServer(server)
	}
}

// AddMediaDir add a new media directory
func (s *Storage) AddMediaDir(directory string) error {
	c := &models.AppConfiguration{}
	err := s.Driver.Read(s.Collections.Configuration, appConfID, c)
	if err != nil {
		return err
	}

	folderPath, err := filepath.Abs(directory)
	if err != nil {
		return err
	}

	mediaDirMap := make(map[string]bool)
	for _, media := range c.ShowsDirs {
		mediaDirMap[media] = true
	}

	if _, ok := mediaDirMap[c.ShowsDir]; !ok {
		c.ShowsDirs = append(c.ShowsDirs, c.ShowsDir)
	}

	if _, ok := mediaDirMap[folderPath]; !ok {
		c.ShowsDirs = append(c.ShowsDirs, folderPath+"/")
	}

	return s.Driver.Write(s.Collections.Configuration, appConfID, c)
}

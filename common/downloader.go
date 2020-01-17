package common

import (
	"errors"
	"fmt"
	"strings"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/services/eztv"
	"github.com/highercomve/couchness/services/showrss"
	"github.com/highercomve/couchness/storage"
	"github.com/highercomve/couchness/utils"
	"github.com/odwrtw/transmission"
)

var (
	showRssService models.FollowService = showrss.New()
	eztvService    models.FollowService = eztv.New()
)

// FollowServices map all types of services
var FollowServices = map[string]models.FollowService{
	showrss.ServiceType: showRssService,
	eztv.ServiceType:    eztvService,
}

// DownloadTorrent use transmission to queue the torrent
func DownloadTorrent(magnetURL, destination string) (*transmission.Torrent, error) {
	auth := strings.Split(storage.AppConfiguration.TransmissionAuth, ":")
	port := storage.AppConfiguration.TransmissionPort
	host := storage.AppConfiguration.TransmissionHost
	conf := transmission.Config{
		Address:  fmt.Sprintf("http://%s:%s/transmission/rpc", host, port),
		User:     auth[0],
		Password: auth[1],
	}
	t, err := transmission.New(conf)
	if err != nil {
		return nil, err
	}

	torrent, err := t.AddTorrent(transmission.AddTorrentArg{
		DownloadDir: destination,
		Filename:    magnetURL,
		Paused:      false,
	})
	if err != nil {
		return nil, err
	}
	return torrent, err
}

// DownloadShow download show depending on show configuration
func DownloadShow(show *models.Show) error {
	switch show.Configuration.FollowType {
	case models.FollowTypeLatest:
		_, err := DownloadLatest(show)
		return err
	case models.FollowTypeSince:
		_, err := downloadSince(show)
		return err
	default:
		return errors.New("Type of show follow not implemented")
	}
}

func downloadSince(show *models.Show) ([]*transmission.Torrent, error) {
	service := FollowServices[show.Configuration.Service]
	sinceNotComplete := true

	begginingOfShow := 101
	page := 1
	limit := 100
	allEpisodes := []*models.TorrentInfo{}
	for sinceNotComplete {
		var s = &models.Show{
			ID:         show.ID,
			ExternalID: show.ExternalID,
			Episodes:   []*models.TorrentInfo{},
		}
		s, err := service.GetShowData(s, page, limit)
		if err != nil {
			return nil, err
		}

		pages := utils.DivCeil(s.TorrentCount, limit)
		lastElement := len(s.Episodes) - 1
		minimalSEN := utils.GetSEN(show.Configuration.Since, 0)
		lastEpisodeSEN := utils.GetSEN(s.Episodes[lastElement].Season, s.Episodes[lastElement].Episode)
		if page < pages && minimalSEN < lastEpisodeSEN && lastEpisodeSEN != begginingOfShow {
			page++
		} else {
			sinceNotComplete = false
		}

		allEpisodes = append(allEpisodes, s.Episodes...)
	}

	downloadedEpisodes := utils.GetEpisodeVersionSince(show.Episodes, show.Configuration.Since, "", "", "")
	eztvEpisodes := utils.GetEpisodeVersionSince(
		allEpisodes,
		show.Configuration.Since,
		show.Configuration.Codec,
		show.Configuration.Resolution,
		show.Configuration.Quality,
	)

	deMap := utils.GetEpisodesMap(downloadedEpisodes)
	missingEpisodes := make([]*transmission.Torrent, 0)
	for _, e := range eztvEpisodes {
		sen := utils.GetSEN(e.Season, e.Episode)
		_, found := deMap[sen]
		if !found {
			fmt.Printf("Downloading Season %d Episode %d \n", e.Season, e.Episode)
			torrent, _ := DownloadTorrent(e.MagnetURL, show.Directory)
			missingEpisodes = append(missingEpisodes, torrent)
		}
	}

	return missingEpisodes, nil
}

// DownloadLatest download last episode if is not already downloaded
func DownloadLatest(show *models.Show) (*transmission.Torrent, error) {
	var s models.Show = *show
	service := FollowServices[show.Configuration.Service]

	newShow, err := service.GetShowData(&s, 1, 10)
	if err != nil {
		return nil, err
	}

	if len(newShow.Episodes) == 0 {
		return nil, errors.New("Show is not on " + service.GetID())
	}

	oldVersions := utils.GetEpisodeVersion(show.Episodes, newShow.Episodes[0].Season, newShow.Episodes[0].Episode, "", "", "")
	if len(oldVersions) == 0 {
		newVersions := utils.GetEpisodeVersion(
			newShow.Episodes,
			newShow.Episodes[0].Season,
			newShow.Episodes[0].Episode,
			newShow.Configuration.Codec,
			newShow.Configuration.Resolution,
			newShow.Configuration.Quality,
		)

		if len(newVersions) == 0 {
			newVersions = utils.GetEpisodeVersion(newShow.Episodes, newShow.Episodes[0].Season, newShow.Episodes[0].Episode, "", "", "")
		}

		if len(newVersions) == 0 {
			return nil, errors.New("Error searching new episode version")
		}

		return DownloadTorrent(newVersions[0].MagnetURL, show.Directory)
	}

	return nil, errors.New("The latest version is already downladed")
}

package common

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/services/eztv"
	"github.com/highercomve/couchness/services/rarbg"
	"github.com/highercomve/couchness/services/showrss"
	"github.com/highercomve/couchness/storage"
	"github.com/highercomve/couchness/utils"
	"github.com/odwrtw/transmission"
)

var (
	showRssService models.FollowService = showrss.New()
	eztvService    models.FollowService = eztv.New()
	rargbService   models.FollowService = rarbg.New()
)

// FollowServices map all types of services
var FollowServices = map[string]models.FollowService{
	showrss.ServiceType: showRssService,
	eztv.ServiceType:    eztvService,
	rarbg.ServiceType:   rargbService,
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
		return errors.New("type of show follow not implemented")
	}
}

// DownloadEpisodeOfShow download show depending on show configuration
func DownloadEpisodeOfShow(show *models.Show, ep string, limit int) (*transmission.Torrent, error) {
	episodes := getShowEpisodesFromServices(show, getShowServices(show), 1, limit)
	if len(episodes) == 0 {
		return nil, errors.New("show is not on show services")
	}

	selectedVersions := []*models.TorrentInfo{}
	i := 1
	for _, episode := range episodes {
		if strings.Contains(strings.ToLower(episode.Location), strings.ToLower(ep)) && !episode.Downloaded {
			selectedVersions = append(
				selectedVersions,
				episode,
			)
			i = i + 1
		}
	}

	if len(selectedVersions) == 0 {
		fmt.Println("Episode not found: ")
		return nil, nil
	}

	fmt.Println("Select the episode you want to download: ")
	for i, episode := range selectedVersions {
		fmt.Printf("%d) %s - size: %s - seeds: %d \n\r", i+1, episode.Title, humanize.Bytes(uint64(episode.Size)), episode.Seeds)
	}

	var input int
	fmt.Scan(&input)

	if input > len(selectedVersions) || input < 1 {
		return nil, fmt.Errorf("the selection most be between %d and 1", len(selectedVersions))
	}

	return DownloadTorrent(selectedVersions[input-1].MagnetURL, show.Directory)
}

func downloadSince(show *models.Show) ([]*transmission.Torrent, error) {
	services := getShowServices(show)

	var allEpisodes models.Episodes
	eztvIndex := findOnSlice(services, eztv.ServiceType)
	if eztvIndex >= 0 {
		eztvEpisodes, err := getTorrentsSince(show, FollowServices[eztv.ServiceType])
		if err != nil {
			return nil, err
		}
		allEpisodes = append(allEpisodes, eztvEpisodes...)
		services = append(services[:eztvIndex], services[eztvIndex+1:]...)
	}
	otherEpisodes := getShowEpisodesFromServices(show, services, 1, 100)

	allEpisodes = append(allEpisodes, otherEpisodes...)
	storage.SortEpisodes(allEpisodes)
	downloadedEpisodes := utils.GetEpisodeVersionSince(show.Episodes, show.Configuration.Since, "", "", "")
	eztvEpisodes := utils.GetMaxSeedsFromList(utils.GetEpisodeVersionSince(
		allEpisodes,
		show.Configuration.Since,
		show.Configuration.Codec,
		show.Configuration.Resolution,
		show.Configuration.Quality,
	))

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
	episodes := getShowEpisodesFromServices(show, getShowServices(show), 1, 30)
	if len(episodes) == 0 {
		return nil, errors.New("show is not on show services")
	}

	storage.SortEpisodes(episodes)
	oldVersions := utils.GetEpisodeVersion(show.Episodes, episodes[0].Season, episodes[0].Episode, "", "", "")
	if len(oldVersions) == 0 {
		newVersions := utils.GetMaxSeedsFromList(utils.GetEpisodeVersion(
			episodes,
			episodes[0].Season,
			episodes[0].Episode,
			show.Configuration.Codec,
			show.Configuration.Resolution,
			show.Configuration.Quality,
		))

		if len(newVersions) == 0 {
			newVersions = utils.GetEpisodeVersion(episodes, episodes[0].Season, episodes[0].Episode, "", "", "")
		}

		if len(newVersions) == 0 {
			return nil, errors.New("error searching new episode version")
		}

		return DownloadTorrent(newVersions[0].MagnetURL, show.Directory)
	}

	return nil, errors.New("the latest version is already downladed")
}

func getTorrentsSince(show *models.Show, service models.FollowService) (models.Episodes, error) {
	sinceNotComplete := true
	begginingOfShowSEN := 101
	minimalSEN := utils.GetSEN(show.Configuration.Since, 1)
	page := 1
	limit := 100
	allEpisodes := []*models.TorrentInfo{}
	for sinceNotComplete {
		s := &models.Show{
			ID:         show.ID,
			ExternalID: show.ExternalID,
			Episodes:   []*models.TorrentInfo{},
		}
		s, err := service.GetShowData(s, page, limit)
		if err != nil {
			return nil, err
		}

		if len(s.Episodes) == 0 {
			sinceNotComplete = false
			continue
		}

		pages := utils.DivCeil(s.TorrentCount, limit)
		lastElement := len(s.Episodes) - 1
		lastEpisodeSEN := utils.GetSEN(s.Episodes[lastElement].Season, s.Episodes[lastElement].Episode)
		if page < pages && lastEpisodeSEN > minimalSEN && lastEpisodeSEN != begginingOfShowSEN {
			page++
		} else {
			sinceNotComplete = false
		}

		allEpisodes = append(allEpisodes, s.Episodes...)
	}

	return allEpisodes, nil
}

func getShowEpisodesFromServices(show *models.Show, services []string, page, limit int) models.Episodes {
	var episodes models.Episodes

	for _, service := range services {
		service := FollowServices[service]
		s, err := service.GetShowData(
			&models.Show{ID: show.ID, ExternalID: show.ExternalID, Episodes: show.Episodes},
			page,
			limit,
		)
		if err != nil {
			fmt.Printf("error downloading from %s", service)
			fmt.Println(err.Error())
		} else {
			episodes = append(episodes, s.Episodes...)
		}
	}

	return episodes
}

func findOnSlice(s []string, searchterm string) int {
	i := sort.SearchStrings(s, searchterm)
	if i < len(s) && s[i] == searchterm {
		return i
	}

	return -1
}

func getShowServices(show *models.Show) []string {
	var services []string
	if len(show.Configuration.Services) > 0 {
		services = show.Configuration.Services
	} else {
		services = []string{show.Configuration.Service}
	}
	return services
}

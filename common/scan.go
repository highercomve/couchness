package common

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar"
	"github.com/gosimple/slug"
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
	"github.com/highercomve/couchness/utils"
)

var defaultConf = &models.ShowConf{
	Service:    "eztv",
	FollowType: "latest",
}

// Scan folder for series
func Scan(folder string, i, r bool) ([]*models.Show, error) {
	matches, err := doublestar.Glob(folder + "/**/*[Ss]*[Ee]*.{mov,avi,wmv,flv,3gp,mp4,mpg,mkv}")
	if err != nil {
		return nil, err
	}
	shows := make(models.ShowsMap, 0)

	for _, m := range matches {
		basename := filepath.Base(m)
		extension := filepath.Ext(m)
		sName := slug.Make(showFolder(m, folder))
		EpisodeData, err := utils.ParseTorrent(basename)
		if err != nil {
			return nil, err
		}
		EpisodeData.Name = sName
		EpisodeData.Extension = extension
		EpisodeData.Location = m
		EpisodeData.Downloaded = true
		var episodes models.Episodes
		if shows[sName] == nil {
			episodes = make(models.Episodes, 0)
		} else {
			episodes = shows[sName].Episodes
		}
		shows[sName] = &models.Show{
			ID:            sName,
			Title:         strings.ReplaceAll(sName, "-", " "),
			Directory:     folder + sName,
			Configuration: defaultConf,
			Episodes:      append(episodes, EpisodeData),
		}
	}

	showArray := make([]*models.Show, 0)
	for _, show := range shows {
		realShow := &models.Show{}
		err := storage.Db.Driver.Read(storage.Db.Collections.Shows, show.ID, realShow)
		if (err != nil && show.ExternalID == "" && i) || r {
			title, id, externalID, err := SearchAndSelectOnImdb(show.Title, "")
			if err != nil {
				fmt.Printf("Error getting imdb information of the show %s \n", show.Title)
				continue
			}
			show.Title = title
			show.ID = id
			show.ExternalID = externalID
			realShow = show
		}
		if err != nil && !i {
			realShow = show
		}
		if err == nil {
			realShow.Episodes = show.Episodes
		}
		storage.NewShowStorage(realShow).Save()
		showArray = append(showArray, realShow)
	}

	return showArray, nil
}

// ScanShowDir loads every episode from a series
func ScanShowDir(dir string, show *models.Show) (*models.Show, error) {
	if show == nil {
		show = &models.Show{
			ID:            getShowDir(dir),
			Directory:     dir,
			Configuration: defaultConf,
		}
	}

	matches, err := doublestar.Glob(dir + "/**/*[Ss]*[Ee]*.{mov,avi,wmv,flv,3gp,mp4,mpg,mkv}")
	if err != nil {
		return nil, err
	}

	show.Episodes = make(models.Episodes, 0)
	for _, m := range matches {
		basename := filepath.Base(m)
		extension := filepath.Ext(m)
		EpisodeData, err := utils.ParseTorrent(basename)
		if err != nil {
			return nil, err
		}

		EpisodeData.Name = show.ID
		EpisodeData.Extension = extension
		EpisodeData.Location = m
		EpisodeData.Downloaded = true

		if show.ExternalID == "" {
			title, id, externalID, err := SearchAndSelectOnImdb(show.Title, "")
			if err != nil {
				show.Title = title
				show.ID = id
				show.ExternalID = externalID
			}
		}
		show.Episodes = append(show.Episodes, EpisodeData)
	}

	return show, nil
}

func showFolder(fullPath, baseFolder string) string {
	f := strings.Split(fullPath, baseFolder)[1]
	f = strings.Split(f, "/")[0]
	return strings.ToLower(f)
}

func getShowDir(p string) string {
	dir := filepath.Dir(p + "/")
	parent := filepath.Base(dir)
	return parent
}

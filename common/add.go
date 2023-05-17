package common

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gosimple/slug"
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
	"github.com/highercomve/couchness/utils"
)

// Add add new show to database
func Add(show *models.Show) (*models.Show, error) {
	show, err := ScanShowDir(show.Directory, show)
	if err != nil {
		return show, err
	}
	err = storage.Db.Driver.Write(storage.Db.Collections.Shows, show.ID, show)
	return show, err
}

// SearchAndSelectOnImdb search and select correct show
func SearchAndSelectOnImdb(title string, typeOf string) (string, string, string, error) {
	externalID := ""
	key := ""
	possibleShows, err := SearchShowInfo(title, typeOf)
	if err != nil {
		return "", "", "", errors.New("Cant find " + title + " on omdb")
	}
	var show OmdbResults
	if possibleShows.TotalResults == "1" {
		show = possibleShows.Search[0]
	} else if len(possibleShows.Search) > 1 {
		fmt.Println("There is more than one result on imdb for your show, please select one and press ENTER: ")

		table := utils.PrintTable([]string{"#", "Name", "Year", "Show ID"}, nil)
		for i, show := range possibleShows.Search {
			table.Append([]string{strconv.Itoa(i + 1), show.Title, show.Year, show.ImdbID})
		}
		table.Render()

		var input int
		fmt.Scan(&input)
		show = possibleShows.Search[input-1]
	} else {
		return "", "", "", errors.New("serie not found on imdb, maybe use the parameter -ex EXTERNAL_ID to set the imdb or showrss ID")
	}

	key = slug.Make(show.Title)
	title = show.Title
	externalID = show.ImdbID

	return title, key, externalID, nil
}

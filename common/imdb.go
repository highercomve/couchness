package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/highercomve/couchness/storage"
)

const omdbAPIURL = "http://www.omdbapi.com/"

// OmdbResponse response from open movie database API
type OmdbResponse struct {
	Search       []OmdbResults
	TotalResults string `json:"totalResults"`
}

// OmdbResults response from open movie database API search results
type OmdbResults struct {
	Title  string `json:"Title"`
	Year   string `json:"Year"`
	ImdbID string `json:"imdbID"`
	Type   string `json:"Type"`
	Poster string `json:"Poster"`
}

// SearchShowInfo Search  show information in omdb
func SearchShowInfo(showName string) (*OmdbResponse, error) {
	query := fmt.Sprintf(
		"?apikey=%s&s=%s&type=series",
		url.QueryEscape(storage.AppConfiguration.OmdbAPIKey),
		url.QueryEscape(showName),
	)
	url := omdbAPIURL + query
	fmt.Printf("Getting Show information from IMDB... \n")
	res, err := http.Get(url)
	if err != nil {
		fmt.Printf("%d", res.StatusCode)
		return nil, err
	}
	results := &OmdbResponse{}
	err = json.NewDecoder(res.Body).Decode(results)
	if err != nil {
		fmt.Printf("%s \n", err.Error())
		return nil, err
	}

	return results, err
}

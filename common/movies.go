package common

import (
	"fmt"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

func GetMovies() (movies []*models.Movie, err error) {
	movies, err = storage.GetAllMovies()
	if err != nil {
		return nil, err
	}

	return movies, nil
}

func AddMovie(movie *models.Movie) (*models.Movie, error) {
	torrents, err := getMovieVersionFromServices(movie, getShowServices(&movie.Show), 1, 50)
	if err != nil {
		return nil, err
	}

	for _, torrent := range torrents {
		fmt.Printf("%++v \n", torrent.Summary())
	}

	// err = storage.Db.Driver.Write(storage.Db.Collections.Movies, movie.ID, movie)
	return movie, err
}

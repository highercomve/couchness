package storage

import (
	"encoding/json"
	"fmt"

	"github.com/highercomve/couchness/models"
)

// MovieStorage storage methods for Movies
type MovieStorage struct {
	*models.Movie
}

// NewMovieStorage create new MovieStorage
func NewMovieStorage(movie *models.Movie) *MovieStorage {
	return &MovieStorage{
		movie,
	}
}

// Save save on database
func (m *MovieStorage) Save() (*MovieStorage, error) {
	err := Db.Driver.Write(Db.Collections.Movies, m.ID, m)
	return m, err
}

// GetAllMovies get all movies
func GetAllMovies() ([]*models.Movie, error) {
	records, err := Db.Driver.ReadAll(Db.Collections.Movies)
	if err != nil {
		return nil, err
	}
	movies := []*models.Movie{}
	for _, f := range records {
		movieFound := &models.Movie{}
		if err := json.Unmarshal([]byte(f), movieFound); err != nil {
			fmt.Println("Error", err)
		}
		movies = append(movies, movieFound)
	}

	return movies, nil
}

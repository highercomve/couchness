package common

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

const sepator = "="

func GetMovies() (movies []*models.Movie, err error) {
	movies, err = storage.GetAllMovies()
	if err != nil {
		return nil, err
	}

	return movies, nil
}

func getTorrents(movie *models.Movie, channel chan<- models.Episodes) {
	torrents, err := getMovieVersionFromServices(movie, getShowServices(&movie.Show), 1, 50, "movies")
	if err != nil {
		channel <- nil
		return
	}

	sort.SliceStable(torrents, func(i, j int) bool {
		return torrents[i].Seeds > torrents[j].Seeds
	})

	channel <- torrents
}

func AddMovie(movie *models.Movie) (*models.Movie, error) {
	fmt.Printf(
		"\n\r\n\r %s Select the movie to download: %s \n\r\n\r",
		strings.Repeat(sepator, 5),
		strings.Repeat(sepator, 5),
	)

	tChannel := make(chan models.Episodes)
	ticker := time.NewTicker(500 * time.Millisecond)
	go getTorrents(movie, tChannel)

	fmt.Print("\033[s")
	var torrents models.Episodes

	i := 0
mainLoop:
	for {
		select {
		case torrents = <-tChannel:
			fmt.Print("\033[u\033[K")
			fmt.Println("")
			break mainLoop
		case <-ticker.C:
			i++
			fmt.Print("\033[u\033[K")
			fmt.Printf("Searching torrents %s", strings.Repeat(".", (i/3)+1))
		}
	}

	fmt.Println("")

	if torrents == nil {
		return nil, fmt.Errorf("no torrents found for movie: %s \n\r", movie.Summary())
	}

	for index, torrent := range torrents {
		fmt.Printf("%d) %++v \n", index+1, torrent.Summary())
	}

	var input int
	fmt.Scan(&input)

	if input > len(torrents) {
		fmt.Printf("Please select between 1 and %d \n\r\n\r", len(torrents))
		return AddMovie(movie)
	}

	torrent := torrents[input-1]
	movie.TorrentInfo = *torrent

	_, err := DownloadTorrent(movie.TorrentInfo.MagnetURL, movie.Directory)
	if err != nil {
		return movie, err
	}

	err = storage.Db.Driver.Write(storage.Db.Collections.Movies, movie.ID, movie)
	if err != nil {
		return movie, err
	}

	return movie, nil
}

package models

import "fmt"

type Movie struct {
	Show
	TorrentInfo
}

// PrintString covert show data into string
func (movie *Movie) PrintString() string {
	return fmt.Sprintf("%s - external ID %s - internal ID %s", movie.Show.Title, movie.ExternalID, movie.ID)
}

// Summary get show summary information
func (movie *Movie) Summary() string {
	return fmt.Sprintf(
		"(ID: %s) %s (size: %d) (IMDB ID: %s)",
		movie.ID,
		movie.Show.Title,
		movie.TorrentInfo.Size,
		movie.ExternalID,
	)
}

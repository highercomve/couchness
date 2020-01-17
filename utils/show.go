package utils

import (
	"strings"

	"github.com/highercomve/couchness/models"
)

// GetEpisodesMap get episodes map by season and episode
func GetEpisodesMap(episodes models.Episodes) map[int]*models.TorrentInfo {
	episodesMap := make(map[int]*models.TorrentInfo, 0)
	for _, e := range episodes {
		sen := GetSEN(e.Season, e.Episode)
		episodesMap[sen] = e
	}
	return episodesMap
}

// GetSEN get sen, SEN is and ID made by the season and the episode
func GetSEN(season, episode int) int {
	return (season * 100) + episode
}

// GetEpisodeVersion Get list of episodes filtering by season, episode, codec, resolution and quality
func GetEpisodeVersion(episodes models.Episodes, sN int, eN int, c, r, q string) models.Episodes {
	eps := make(models.Episodes, 0)
	for _, e := range episodes {
		eC := strings.ToLower(e.Codec)
		eR := strings.ToLower(e.Resolution)
		eQ := strings.ToLower(e.Quality)

		if (sN == -1 || e.Season == sN) && (eN == -1 || e.Episode == eN) && (c == "" || eC == c) && (r == "" || eR == r) && (q == "" || eQ == q) {
			eps = append(eps, e)
		}
	}

	return eps
}

// GetEpisodeVersionSince Get list of episodes filtering since season with codec, resolution and quality
func GetEpisodeVersionSince(episodes models.Episodes, sN int, c, r, q string) models.Episodes {
	eps := make(models.Episodes, 0)
	for _, e := range episodes {
		eC := strings.ToLower(e.Codec)
		eR := strings.ToLower(e.Resolution)
		eQ := strings.ToLower(e.Quality)

		if e.Season >= sN && (c == "" || eC == c) && (r == "" || eR == r) && (q == "" || eQ == q) {
			eps = append(eps, e)
		}
	}

	return eps
}

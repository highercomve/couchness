package utils

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/highercomve/couchness/models"
	"github.com/olekukonko/tablewriter"
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

		if os.Getenv("DEBUG") != "" {
			fmt.Printf("Minimun S=%d c=%s,r=%s,q=%s. \n", sN, c, r, q)
			fmt.Printf("Compared with S%dE%d c=%s,r=%s,q=%s.\n", e.Season, e.Episode, eC, eR, eQ)
		}

		if e.Season >= sN && (c == "" || eC == c) && (r == "" || eR == r) && (q == "" || eQ == q) {
			eps = append(eps, e)
		}
	}

	return eps
}

// GetMinimunSizeFromList get minimum size episodes from a list
func GetMinimunSizeFromList(episodes models.Episodes) models.Episodes {
	episodesMap := make(map[int]*models.TorrentInfo, 0)
	for _, e := range episodes {
		sen := GetSEN(e.Season, e.Episode)
		lastE, found := episodesMap[sen]
		if found && e.Size > lastE.Size {
			continue
		}
		episodesMap[sen] = e
	}
	newEpisodes := make(models.Episodes, 0)
	for _, e := range episodesMap {
		newEpisodes = append(newEpisodes, e)
	}
	sort.SliceStable(newEpisodes, func(i, j int) bool {
		se1 := GetSEN(newEpisodes[i].Season, newEpisodes[i].Episode)
		se2 := GetSEN(newEpisodes[j].Season, newEpisodes[j].Episode)

		return se1 > se2
	})
	return newEpisodes
}

// GetMaxSeedsFromList get maximun amount of seeds episodes from a list
func GetMaxSeedsFromList(episodes models.Episodes) models.Episodes {
	episodesMap := make(map[int]*models.TorrentInfo, 0)
	for _, e := range episodes {
		sen := GetSEN(e.Season, e.Episode)
		lastE, found := episodesMap[sen]
		if found && e.Seeds < lastE.Seeds {
			continue
		}
		episodesMap[sen] = e
	}
	newEpisodes := make(models.Episodes, 0)
	for _, e := range episodesMap {
		newEpisodes = append(newEpisodes, e)
	}
	sort.SliceStable(newEpisodes, func(i, j int) bool {
		se1 := GetSEN(newEpisodes[i].Season, newEpisodes[i].Episode)
		se2 := GetSEN(newEpisodes[j].Season, newEpisodes[j].Episode)

		return se1 > se2
	})
	return newEpisodes
}

func PrintTable(header []string, values [][]string) *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetColumnSeparator(" ")
	table.SetHeader(header)

	for _, value := range values {
		table.Append(value)
	}

	if len(values) > 0 {
		table.Render()
	}

	return table
}

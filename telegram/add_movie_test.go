package telegram

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/highercomve/couchness/common"
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

func TestAddMovieKeyboardsFitTelegramLimits(t *testing.T) {
	session := &addMovieSession{
		ID: strings.Repeat("x", 24),
		Results: []common.OmdbResults{
			{Title: strings.Repeat("Long movie title ", 20), Year: "2026"},
		},
		Torrents: models.Episodes{
			{Title: strings.Repeat("Long torrent title ", 20), Size: 1024, Seeds: 42},
		},
	}
	for _, markup := range []InlineKeyboardMarkup{session.movieResultsKeyboard(), session.movieTorrentsKeyboard()} {
		for _, row := range markup.InlineKeyboard {
			for _, item := range row {
				if len([]rune(item.Text)) > 60 {
					t.Fatalf("button label too long: %d", len([]rune(item.Text)))
				}
				if len([]byte(item.CallbackData)) > 64 {
					t.Fatalf("callback data too long: %d", len([]byte(item.CallbackData)))
				}
			}
		}
	}
}

func TestAddMovieWizardQueuesAndPersistsSelection(t *testing.T) {
	directory, err := ioutil.TempDir("", "couchness-add-movie-wizard-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)

	originalDB := storage.Db
	originalConfiguration := storage.AppConfiguration
	defer func() {
		storage.Db = originalDB
		storage.AppConfiguration = originalConfiguration
	}()
	storage.Db, err = storage.New(filepath.Join(directory, "database"), nil)
	if err != nil {
		t.Fatal(err)
	}
	storage.AppConfiguration = &models.AppConfiguration{MoviesDir: filepath.Join(directory, "movies")}
	configuration := &models.TelegramConfiguration{
		Enabled: true,
		Users: map[string]*models.TelegramUser{
			"42": {ID: 42, Role: models.TelegramRoleOwner},
		},
		Invites: make(map[string]*models.TelegramInvite),
	}
	if err := storage.SaveTelegramConfiguration(configuration); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(request.URL.Path, "/sendMessage") {
			fmt.Fprint(writer, `{"ok":true,"result":{"message_id":99}}`)
			return
		}
		fmt.Fprint(writer, `{"ok":true,"result":true}`)
	}))
	defer server.Close()

	searchType := ""
	queued := false
	bot := &Bot{
		client:        &apiClient{token: "test", baseURL: server.URL, httpClient: &http.Client{Timeout: time.Second}},
		sessions:      make(map[int64]*addShowSession),
		movieSessions: make(map[int64]*addMovieSession),
		removals:      make(map[int64]*removeShowSession),
		searchShows: func(title, typeOf string) (*common.OmdbResponse, error) {
			searchType = typeOf
			return &common.OmdbResponse{Search: []common.OmdbResults{{Title: "The Matrix", Year: "1999", ImdbID: "tt0133093"}}}, nil
		},
		searchMovieTorrents: func(movie *models.Movie) (models.Episodes, error) {
			if movie.ID != "the-matrix-1999" || movie.ExternalID != "tt0133093" {
				t.Fatalf("unexpected movie search: %#v", movie)
			}
			return models.Episodes{{Title: "The Matrix 1999 1080p", MagnetURL: "magnet:?xt=test", Size: 2048, Seeds: 99}}, nil
		},
		downloadMovie: func(movie *models.Movie, torrent *models.TorrentInfo) error {
			queued = true
			movie.TorrentInfo = *torrent
			movie.TorrentInfo.Downloaded = true
			_, err := storage.NewMovieStorage(movie).Save()
			return err
		},
	}
	message := &Message{MessageID: 1, Text: "/add_movie Matrix", From: User{ID: 42}, Chat: Chat{ID: 42, Type: "private"}}
	bot.handleMessage(context.Background(), message)
	session := bot.movieSessions[42]
	if session == nil || session.MessageID != 99 || searchType != "movie" {
		t.Fatalf("movie wizard session not started: %#v, search type %q", session, searchType)
	}

	callback := func(action string) {
		bot.handleCallback(context.Background(), &CallbackQuery{
			ID:      "callback-" + action,
			From:    User{ID: 42},
			Message: &Message{MessageID: 99, Chat: Chat{ID: 42, Type: "private"}},
			Data:    session.callback(action),
		})
	}
	callback("select_result|0")
	callback("select_torrent|0")
	callback("confirm")

	if !queued {
		t.Fatal("movie was not queued")
	}
	if bot.movieSessions[42] != nil {
		t.Fatal("movie session was not closed")
	}
	movies, err := common.GetMovies()
	if err != nil {
		t.Fatal(err)
	}
	if len(movies) != 1 || movies[0].ID != "the-matrix-1999" || !movies[0].Downloaded {
		t.Fatalf("unexpected saved movies: %#v", movies)
	}
}

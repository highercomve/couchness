package telegram

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/highercomve/couchness/common"
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
	"github.com/highercomve/couchness/utils/humanize"
)

type addMovieSession struct {
	ID        string
	UserID    int64
	ChatID    int64
	MessageID int64
	Results   []common.OmdbResults
	Selected  common.OmdbResults
	Movie     *models.Movie
	Torrents  models.Episodes
	Torrent   *models.TorrentInfo
	ExpiresAt time.Time
}

func (bot *Bot) startAddMovie(ctx context.Context, message *Message, user *models.TelegramUser, arguments []string) {
	if !isAdministrator(user.Role) {
		bot.reply(ctx, message.Chat.ID, "Only owners and admins can add movies.")
		return
	}
	title := strings.TrimSpace(strings.Join(arguments, " "))
	if title == "" {
		bot.reply(ctx, message.Chat.ID, "Send a title after the command.\nExample: /add_movie The Matrix")
		return
	}
	if storage.AppConfiguration == nil || strings.TrimSpace(storage.AppConfiguration.MoviesDir) == "" {
		bot.reply(ctx, message.Chat.ID, "No movies directory configured. Set COUCHNESS_MOVIES_DIR first.")
		return
	}

	_ = bot.client.sendChatAction(ctx, message.Chat.ID, "typing")
	results, err := bot.searchShows(title, "movie")
	if err != nil {
		if results != nil && len(results.Search) == 0 {
			bot.reply(ctx, message.Chat.ID, "No matching movies found. Check spelling and try again.")
			return
		}
		bot.reply(ctx, message.Chat.ID, "OMDb search failed: "+err.Error())
		return
	}
	if len(results.Search) == 0 {
		bot.reply(ctx, message.Chat.ID, "No matching movies found. Check spelling and try again.")
		return
	}

	limit := len(results.Search)
	if limit > 8 {
		limit = 8
	}
	code, err := randomCode()
	if err != nil {
		bot.reply(ctx, message.Chat.ID, "Could not start movie setup: "+err.Error())
		return
	}
	session := &addMovieSession{
		ID:        code,
		UserID:    message.From.ID,
		ChatID:    message.Chat.ID,
		Results:   results.Search[:limit],
		ExpiresAt: time.Now().UTC().Add(addShowSessionLifetime),
	}
	if bot.movieSessions == nil {
		bot.movieSessions = make(map[int64]*addMovieSession)
	}
	now := time.Now().UTC()
	for userID, active := range bot.movieSessions {
		if now.After(active.ExpiresAt) {
			delete(bot.movieSessions, userID)
		}
	}
	delete(bot.sessions, message.From.ID)
	delete(bot.removals, message.From.ID)
	bot.movieSessions[message.From.ID] = session

	sent, err := bot.client.sendMessageWithKeyboard(ctx, message.Chat.ID, "Select the correct movie:", session.movieResultsKeyboard())
	if err != nil {
		delete(bot.movieSessions, message.From.ID)
		bot.reply(ctx, message.Chat.ID, "Could not show movie results: "+err.Error())
		return
	}
	session.MessageID = sent.MessageID
}

func (bot *Bot) handleAddMovieCallback(ctx context.Context, callback *CallbackQuery, parts []string) {
	session := bot.movieSessions[callback.From.ID]
	if session == nil || session.ID != parts[1] || time.Now().UTC().After(session.ExpiresAt) {
		delete(bot.movieSessions, callback.From.ID)
		bot.reply(ctx, callback.Message.Chat.ID, "This setup expired. Start again with /add_movie <title>.")
		return
	}
	session.MessageID = callback.Message.MessageID
	session.ExpiresAt = time.Now().UTC().Add(addShowSessionLifetime)

	switch parts[2] {
	case "select_result":
		bot.selectMovie(ctx, session, parts)
	case "select_torrent":
		bot.selectMovieTorrent(ctx, session, parts)
	case "back_results":
		session.Movie = nil
		session.Torrents = nil
		session.Torrent = nil
		bot.editMovieSession(ctx, session, "Select the correct movie:", session.movieResultsKeyboard())
	case "back_torrents":
		bot.renderMovieTorrents(ctx, session)
	case "confirm":
		bot.confirmAddMovie(ctx, session)
	case "cancel", "done":
		delete(bot.movieSessions, callback.From.ID)
		bot.editMovieSession(ctx, session, "Movie setup closed.", InlineKeyboardMarkup{})
	}
}

func (bot *Bot) selectMovie(ctx context.Context, session *addMovieSession, parts []string) {
	if len(parts) != 4 {
		return
	}
	index, err := strconv.Atoi(parts[3])
	if err != nil || index < 0 || index >= len(session.Results) {
		return
	}
	session.Selected = session.Results[index]
	directory, err := filepath.Abs(storage.AppConfiguration.MoviesDir)
	if err != nil {
		bot.editMovieSession(ctx, session, "Could not resolve movies directory: "+err.Error(), session.movieResultsKeyboard())
		return
	}
	session.Movie = &models.Movie{Show: models.Show{
		ID:         slug.Make(strings.TrimSpace(session.Selected.Title + " " + session.Selected.Year)),
		Title:      session.Selected.Title,
		ExternalID: session.Selected.ImdbID,
		Directory:  directory + "/",
		Configuration: &models.ShowConf{
			Services: append([]string(nil), common.DefaultMovieServices...),
		},
	}}

	if movies, listErr := common.GetMovies(); listErr == nil {
		for _, movie := range movies {
			if movie.ExternalID == session.Movie.ExternalID || movie.ID == session.Movie.ID {
				session.Movie = movie
				break
			}
		}
	}

	bot.editMovieSession(ctx, session, "Searching torrents for "+session.Selected.Title+"…", InlineKeyboardMarkup{})
	search := bot.searchMovieTorrents
	if search == nil {
		search = common.SearchMovieTorrents
	}
	torrents, err := search(session.Movie)
	if err != nil {
		bot.editMovieSession(ctx, session, "Could not find movie torrents: "+err.Error(), keyboard(
			row(button("Try again", session.callback("select_result|"+strconv.Itoa(index)))),
			row(button("Back", session.callback("back_results")), button("Cancel", session.callback("cancel"))),
		))
		return
	}
	if len(torrents) > 8 {
		torrents = torrents[:8]
	}
	session.Torrents = torrents
	bot.renderMovieTorrents(ctx, session)
}

func (bot *Bot) selectMovieTorrent(ctx context.Context, session *addMovieSession, parts []string) {
	if len(parts) != 4 {
		return
	}
	index, err := strconv.Atoi(parts[3])
	if err != nil || index < 0 || index >= len(session.Torrents) {
		return
	}
	torrent := session.Torrents[index]
	session.Torrent = torrent
	message := fmt.Sprintf(
		"Queue this movie?\n\nTitle: %s (%s)\nTorrent: %s\nSize: %s\nSeeds: %d",
		session.Selected.Title,
		session.Selected.Year,
		movieTorrentTitle(torrent),
		humanize.Bytes(uint64(torrent.Size)),
		torrent.Seeds,
	)
	keyboard := keyboard(
		row(button("Queue movie", session.callback("confirm"))),
		row(button("Back", session.callback("back_torrents")), button("Cancel", session.callback("cancel"))),
	)
	bot.editMovieSession(ctx, session, message, keyboard)
}

func (bot *Bot) renderMovieTorrents(ctx context.Context, session *addMovieSession) {
	message := fmt.Sprintf("%s (%s)\n\nSelect a torrent:", session.Selected.Title, session.Selected.Year)
	bot.editMovieSession(ctx, session, message, session.movieTorrentsKeyboard())
}

func (bot *Bot) confirmAddMovie(ctx context.Context, session *addMovieSession) {
	if session.Movie == nil || session.Torrent == nil {
		return
	}
	bot.editMovieSession(ctx, session, "Queueing "+session.Movie.Show.Title+" in Transmission…", InlineKeyboardMarkup{})
	download := bot.downloadMovie
	if download == nil {
		download = common.DownloadMovie
	}
	if err := download(session.Movie, session.Torrent); err != nil {
		message := "Could not queue movie: " + err.Error()
		markup := keyboard(
			row(button("Retry", session.callback("confirm"))),
			row(button("Back", session.callback("back_torrents")), button("Cancel", session.callback("cancel"))),
		)
		bot.editMovieSession(ctx, session, message, markup)
		return
	}
	delete(bot.movieSessions, session.UserID)
	bot.editMovieSession(ctx, session, session.Movie.Show.Title+" queued and added to the movies list.", InlineKeyboardMarkup{})
}

func (bot *Bot) editMovieSession(ctx context.Context, session *addMovieSession, message string, markup InlineKeyboardMarkup) {
	_ = bot.client.editMessage(ctx, session.ChatID, session.MessageID, message, markup)
}

func (session *addMovieSession) callback(action string) string {
	return "movie|" + session.ID + "|" + action
}

func (session *addMovieSession) movieResultsKeyboard() InlineKeyboardMarkup {
	rows := make([][]InlineKeyboardButton, 0, len(session.Results)+1)
	for index, result := range session.Results {
		label := truncateButtonText(fmt.Sprintf("%s (%s)", result.Title, result.Year), 60)
		rows = append(rows, row(button(label, session.callback("select_result|"+strconv.Itoa(index)))))
	}
	rows = append(rows, row(button("Cancel", session.callback("cancel"))))
	return keyboard(rows...)
}

func (session *addMovieSession) movieTorrentsKeyboard() InlineKeyboardMarkup {
	rows := make([][]InlineKeyboardButton, 0, len(session.Torrents)+2)
	for index, torrent := range session.Torrents {
		label := fmt.Sprintf("%s · %s · %d seeds", movieTorrentTitle(torrent), humanize.Bytes(uint64(torrent.Size)), torrent.Seeds)
		rows = append(rows, row(button(truncateButtonText(label, 60), session.callback("select_torrent|"+strconv.Itoa(index)))))
	}
	rows = append(rows, row(button("Back", session.callback("back_results")), button("Cancel", session.callback("cancel"))))
	return keyboard(rows...)
}

func movieTorrentTitle(torrent *models.TorrentInfo) string {
	if strings.TrimSpace(torrent.Title) != "" {
		return torrent.Title
	}
	return torrent.Name
}

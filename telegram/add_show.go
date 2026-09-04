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
	"github.com/highercomve/couchness/services/eztv"
	"github.com/highercomve/couchness/storage"
)

const addShowSessionLifetime = 15 * time.Minute

type addShowSession struct {
	ID             string
	UserID         int64
	ChatID         int64
	MessageID      int64
	Results        []common.OmdbResults
	Selected       common.OmdbResults
	FollowType     string
	Since          int
	Resolution     string
	ShowID         string
	AwaitingSeason bool
	Added          bool
	ExpiresAt      time.Time
}

func (bot *Bot) startAddShow(ctx context.Context, message *Message, user *models.TelegramUser, arguments []string) {
	if !isAdministrator(user.Role) {
		bot.reply(ctx, message.Chat.ID, "Only owners and admins can add shows.")
		return
	}
	title := strings.TrimSpace(strings.Join(arguments, " "))
	if title == "" {
		bot.reply(ctx, message.Chat.ID, "Send a title after the command.\nExample: /add_show The Last of Us")
		return
	}

	_ = bot.client.sendChatAction(ctx, message.Chat.ID, "typing")
	results, err := bot.searchShows(title, "series")
	if err != nil {
		if results != nil && len(results.Search) == 0 {
			bot.reply(ctx, message.Chat.ID, "No matching shows found. Check spelling and try again.")
			return
		}
		bot.reply(ctx, message.Chat.ID, "OMDb search failed: "+err.Error())
		return
	}
	if len(results.Search) == 0 {
		bot.reply(ctx, message.Chat.ID, "No matching shows found. Check spelling and try again.")
		return
	}

	limit := len(results.Search)
	if limit > 8 {
		limit = 8
	}
	code, err := randomCode()
	if err != nil {
		bot.reply(ctx, message.Chat.ID, "Could not start setup: "+err.Error())
		return
	}
	session := &addShowSession{
		ID:        code,
		UserID:    message.From.ID,
		ChatID:    message.Chat.ID,
		Results:   results.Search[:limit],
		ExpiresAt: time.Now().UTC().Add(addShowSessionLifetime),
	}
	now := time.Now().UTC()
	for userID, active := range bot.sessions {
		if now.After(active.ExpiresAt) {
			delete(bot.sessions, userID)
		}
	}
	delete(bot.removals, message.From.ID)
	delete(bot.movieSessions, message.From.ID)
	bot.sessions[message.From.ID] = session

	sent, err := bot.client.sendMessageWithKeyboard(ctx, message.Chat.ID, "Select the correct show:", session.resultsKeyboard())
	if err != nil {
		delete(bot.sessions, message.From.ID)
		bot.reply(ctx, message.Chat.ID, "Could not show search results: "+err.Error())
		return
	}
	session.MessageID = sent.MessageID
}

func (bot *Bot) handleCallback(ctx context.Context, callback *CallbackQuery) {
	_ = bot.client.answerCallback(ctx, callback.ID, "")
	if callback.Message == nil || callback.Message.Chat.Type != "private" {
		return
	}

	configuration, err := storage.GetTelegramConfiguration()
	if err != nil || !configuration.Enabled {
		return
	}
	user := configuration.Users[strconv.FormatInt(callback.From.ID, 10)]
	if user == nil || !isAdministrator(user.Role) {
		bot.reply(ctx, callback.Message.Chat.ID, "Permission denied.")
		return
	}

	parts := strings.Split(callback.Data, "|")
	if len(parts) < 3 {
		return
	}
	switch parts[0] {
	case "add":
		bot.handleAddShowCallback(ctx, callback, parts)
	case "remove":
		bot.handleRemoveShowCallback(ctx, callback, parts)
	case "movie":
		bot.handleAddMovieCallback(ctx, callback, parts)
	}
}

func (bot *Bot) handleAddShowCallback(ctx context.Context, callback *CallbackQuery, parts []string) {
	session := bot.sessions[callback.From.ID]
	if session == nil || session.ID != parts[1] || time.Now().UTC().After(session.ExpiresAt) {
		delete(bot.sessions, callback.From.ID)
		bot.reply(ctx, callback.Message.Chat.ID, "This setup expired. Start again with /add_show <title>.")
		return
	}
	session.MessageID = callback.Message.MessageID
	session.ExpiresAt = time.Now().UTC().Add(addShowSessionLifetime)
	action := parts[2]

	switch action {
	case "select":
		bot.selectShow(ctx, session, parts)
	case "follow_latest":
		session.FollowType = models.FollowTypeLatest
		session.Since = 0
		session.AwaitingSeason = false
		bot.renderResolution(ctx, session)
	case "follow_since":
		session.FollowType = models.FollowTypeSince
		session.AwaitingSeason = true
		bot.renderSeason(ctx, session)
	case "resolution_any", "resolution_720p", "resolution_1080p", "resolution_2160p":
		session.Resolution = strings.TrimPrefix(action, "resolution_")
		if session.Resolution == "any" {
			session.Resolution = ""
		}
		bot.renderConfirmation(ctx, session)
	case "back_results":
		session.AwaitingSeason = false
		bot.editSession(ctx, session, "Select the correct show:", session.resultsKeyboard())
	case "back_follow":
		session.AwaitingSeason = false
		bot.renderFollowType(ctx, session)
	case "back_resolution":
		bot.renderResolution(ctx, session)
	case "confirm":
		bot.confirmAddShow(ctx, session)
	case "prepare_download":
		bot.renderDownloadConfirmation(ctx, session)
	case "download":
		bot.downloadAddedShow(ctx, session)
	case "cancel", "done":
		delete(bot.sessions, callback.From.ID)
		bot.editSession(ctx, session, "Show setup closed.", InlineKeyboardMarkup{})
	}
}

func (bot *Bot) selectShow(ctx context.Context, session *addShowSession, parts []string) {
	if len(parts) != 4 {
		return
	}
	index, err := strconv.Atoi(parts[3])
	if err != nil || index < 0 || index >= len(session.Results) {
		return
	}
	session.Selected = session.Results[index]
	session.ShowID = slug.Make(session.Selected.Title)

	shows, err := common.GetShows()
	if err == nil {
		for _, show := range shows {
			if show.ExternalID == session.Selected.ImdbID || show.ID == session.ShowID {
				session.ShowID = show.ID
				session.Added = true
				if show.Configuration != nil {
					session.FollowType = show.Configuration.FollowType
					session.Since = show.Configuration.Since
					session.Resolution = show.Configuration.Resolution
				}
				bot.renderAlreadyAdded(ctx, session)
				return
			}
		}
	}
	bot.renderFollowType(ctx, session)
}

func (bot *Bot) handleSessionText(ctx context.Context, message *Message, user *models.TelegramUser) {
	session := bot.sessions[message.From.ID]
	if session == nil || !session.AwaitingSeason || !isAdministrator(user.Role) {
		return
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		delete(bot.sessions, message.From.ID)
		bot.reply(ctx, message.Chat.ID, "This setup expired. Start again with /add_show <title>.")
		return
	}
	season, err := strconv.Atoi(strings.TrimSpace(message.Text))
	if err != nil || season < 1 || season > 99 {
		bot.reply(ctx, message.Chat.ID, "Send a season number from 1 to 99, or use /cancel.")
		return
	}
	session.Since = season
	session.AwaitingSeason = false
	session.ExpiresAt = time.Now().UTC().Add(addShowSessionLifetime)
	bot.renderResolution(ctx, session)
}

func (bot *Bot) cancelSession(ctx context.Context, chatID, userID int64) {
	session := bot.sessions[userID]
	if session != nil {
		delete(bot.sessions, userID)
		bot.editSession(ctx, session, "Show setup cancelled.", InlineKeyboardMarkup{})
		return
	}
	removal := bot.removals[userID]
	if removal != nil {
		delete(bot.removals, userID)
		bot.editRemoval(ctx, removal, "Show removal cancelled.", InlineKeyboardMarkup{})
		return
	}
	movie := bot.movieSessions[userID]
	if movie != nil {
		delete(bot.movieSessions, userID)
		bot.editMovieSession(ctx, movie, "Movie setup cancelled.", InlineKeyboardMarkup{})
		return
	}
	bot.reply(ctx, chatID, "No active setup.")
}

func (bot *Bot) renderFollowType(ctx context.Context, session *addShowSession) {
	message := fmt.Sprintf("%s (%s)\n\nWhich episodes should Couchness follow?", session.Selected.Title, session.Selected.Year)
	keyboard := keyboard(
		row(button("Latest episode", session.callback("follow_latest"))),
		row(button("From a season", session.callback("follow_since"))),
		row(button("Back", session.callback("back_results")), button("Cancel", session.callback("cancel"))),
	)
	bot.editSession(ctx, session, message, keyboard)
}

func (bot *Bot) renderSeason(ctx context.Context, session *addShowSession) {
	message := fmt.Sprintf("%s\n\nSend the first season to follow, from 1 to 99.", session.Selected.Title)
	keyboard := keyboard(row(button("Back", session.callback("back_follow")), button("Cancel", session.callback("cancel"))))
	bot.editSession(ctx, session, message, keyboard)
}

func (bot *Bot) renderResolution(ctx context.Context, session *addShowSession) {
	message := fmt.Sprintf("%s\n\nPreferred resolution? If unavailable, Couchness falls back to another version.", session.Selected.Title)
	keyboard := keyboard(
		row(button("Any", session.callback("resolution_any")), button("720p", session.callback("resolution_720p"))),
		row(button("1080p", session.callback("resolution_1080p")), button("2160p", session.callback("resolution_2160p"))),
		row(button("Back", session.callback("back_follow")), button("Cancel", session.callback("cancel"))),
	)
	bot.editSession(ctx, session, message, keyboard)
}

func (bot *Bot) renderConfirmation(ctx context.Context, session *addShowSession) {
	follow := "Latest episode"
	if session.FollowType == models.FollowTypeSince {
		follow = fmt.Sprintf("From season %d", session.Since)
	}
	resolution := session.Resolution
	if resolution == "" {
		resolution = "Any"
	}
	message := fmt.Sprintf("Add this show?\n\nTitle: %s (%s)\nFollow: %s\nResolution: %s\nService: EZTV", session.Selected.Title, session.Selected.Year, follow, resolution)
	keyboard := keyboard(
		row(button("Add show", session.callback("confirm"))),
		row(button("Back", session.callback("back_resolution")), button("Cancel", session.callback("cancel"))),
	)
	bot.editSession(ctx, session, message, keyboard)
}

func (bot *Bot) confirmAddShow(ctx context.Context, session *addShowSession) {
	bot.editSession(ctx, session, "Adding "+session.Selected.Title+"…", InlineKeyboardMarkup{})
	directory := filepath.Join(storage.AppConfiguration.ShowsDir, session.ShowID) + "/"
	show := &models.Show{
		ID:         session.ShowID,
		Title:      session.Selected.Title,
		ExternalID: session.Selected.ImdbID,
		Directory:  directory,
		Configuration: &models.ShowConf{
			Services:   []string{eztv.ServiceType},
			FollowType: session.FollowType,
			Since:      session.Since,
			Resolution: session.Resolution,
		},
	}
	if _, err := common.Add(show); err != nil {
		bot.editSession(ctx, session, "Could not add show: "+err.Error(), keyboard(row(button("Try again", session.callback("confirm")), button("Cancel", session.callback("cancel")))))
		return
	}
	session.Added = true
	message := fmt.Sprintf("%s added successfully.\nID: %s", show.Title, show.ID)
	keyboard := keyboard(row(button("Download now", session.callback("prepare_download"))), row(button("Done", session.callback("done"))))
	bot.editSession(ctx, session, message, keyboard)
}

func (bot *Bot) renderAlreadyAdded(ctx context.Context, session *addShowSession) {
	message := fmt.Sprintf("%s is already in your library.\nID: %s", session.Selected.Title, session.ShowID)
	keyboard := keyboard(
		row(button("Download", session.callback("prepare_download"))),
		row(button("Back", session.callback("back_results")), button("Close", session.callback("done"))),
	)
	bot.editSession(ctx, session, message, keyboard)
}

func (bot *Bot) renderDownloadConfirmation(ctx context.Context, session *addShowSession) {
	message := "Queue latest available episode in Transmission?"
	if session.FollowType == models.FollowTypeSince {
		message = fmt.Sprintf("Queue every missing episode from season %d? This may add multiple torrents.", session.Since)
	}
	keyboard := keyboard(
		row(button("Queue download", session.callback("download"))),
		row(button("Not now", session.callback("done"))),
	)
	bot.editSession(ctx, session, message, keyboard)
}

func (bot *Bot) downloadAddedShow(ctx context.Context, session *addShowSession) {
	if !session.Added || session.ShowID == "" {
		return
	}
	bot.editSession(ctx, session, "Searching for an episode to download…", InlineKeyboardMarkup{})
	if err := common.Download(session.ShowID); err != nil {
		message := "Could not queue download: " + err.Error()
		keyboard := keyboard(row(button("Retry", session.callback("download"))), row(button("Done", session.callback("done"))))
		bot.editSession(ctx, session, message, keyboard)
		return
	}
	delete(bot.sessions, session.UserID)
	bot.editSession(ctx, session, "Download queued in Transmission.", InlineKeyboardMarkup{})
}

func (bot *Bot) editSession(ctx context.Context, session *addShowSession, message string, markup InlineKeyboardMarkup) {
	_ = bot.client.editMessage(ctx, session.ChatID, session.MessageID, message, markup)
}

func (session *addShowSession) callback(action string) string {
	return "add|" + session.ID + "|" + action
}

func (session *addShowSession) resultsKeyboard() InlineKeyboardMarkup {
	rows := make([][]InlineKeyboardButton, 0, len(session.Results)+1)
	for index, result := range session.Results {
		label := truncateButtonText(fmt.Sprintf("%s (%s)", result.Title, result.Year), 60)
		rows = append(rows, row(button(label, session.callback("select|"+strconv.Itoa(index)))))
	}
	rows = append(rows, row(button("Cancel", session.callback("cancel"))))
	return keyboard(rows...)
}

func keyboard(rows ...[]InlineKeyboardButton) InlineKeyboardMarkup {
	return InlineKeyboardMarkup{InlineKeyboard: rows}
}

func row(buttons ...InlineKeyboardButton) []InlineKeyboardButton {
	return buttons
}

func button(text, callback string) InlineKeyboardButton {
	return InlineKeyboardButton{Text: text, CallbackData: callback}
}

func truncateButtonText(value string, maximum int) string {
	runes := []rune(value)
	if len(runes) <= maximum {
		return value
	}
	return string(runes[:maximum-1]) + "…"
}

package telegram

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/highercomve/couchness/common"
	"github.com/highercomve/couchness/models"
)

const removeShowPageSize = 6

type removeShowSession struct {
	ID        string
	UserID    int64
	ChatID    int64
	MessageID int64
	Shows     []*models.Show
	Selected  *models.Show
	Page      int
	ExpiresAt time.Time
}

func (bot *Bot) startRemoveShow(ctx context.Context, message *Message, user *models.TelegramUser, arguments []string) {
	if !isAdministrator(user.Role) {
		bot.reply(ctx, message.Chat.ID, "Only owners and admins can remove shows.")
		return
	}
	shows, err := common.GetShows()
	if err != nil {
		bot.reply(ctx, message.Chat.ID, "Could not list shows: "+err.Error())
		return
	}
	if len(shows) == 0 {
		bot.reply(ctx, message.Chat.ID, "No shows to remove.")
		return
	}
	sort.SliceStable(shows, func(i, j int) bool {
		return strings.ToLower(shows[i].Title) < strings.ToLower(shows[j].Title)
	})

	code, err := randomCode()
	if err != nil {
		bot.reply(ctx, message.Chat.ID, "Could not start removal: "+err.Error())
		return
	}
	session := &removeShowSession{
		ID:        code,
		UserID:    message.From.ID,
		ChatID:    message.Chat.ID,
		Shows:     shows,
		ExpiresAt: time.Now().UTC().Add(addShowSessionLifetime),
	}
	if len(arguments) > 0 {
		showID := strings.TrimSpace(arguments[0])
		for _, show := range shows {
			if show.ID == showID {
				session.Selected = show
				break
			}
		}
		if session.Selected == nil {
			bot.reply(ctx, message.Chat.ID, "Show not found. Use /remove_show to choose from your library.")
			return
		}
	}

	if bot.removals == nil {
		bot.removals = make(map[int64]*removeShowSession)
	}
	delete(bot.sessions, message.From.ID)
	delete(bot.movieSessions, message.From.ID)
	bot.removals[message.From.ID] = session
	text := session.listText()
	markup := session.listKeyboard()
	if session.Selected != nil {
		text = removalConfirmation(session.Selected)
		markup = session.confirmationKeyboard()
	}
	sent, err := bot.client.sendMessageWithKeyboard(ctx, message.Chat.ID, text, markup)
	if err != nil {
		delete(bot.removals, message.From.ID)
		bot.reply(ctx, message.Chat.ID, "Could not start removal: "+err.Error())
		return
	}
	session.MessageID = sent.MessageID
}

func (bot *Bot) handleRemoveShowCallback(ctx context.Context, callback *CallbackQuery, parts []string) {
	session := bot.removals[callback.From.ID]
	if session == nil || session.ID != parts[1] || time.Now().UTC().After(session.ExpiresAt) {
		delete(bot.removals, callback.From.ID)
		bot.reply(ctx, callback.Message.Chat.ID, "This removal expired. Start again with /remove_show.")
		return
	}
	session.MessageID = callback.Message.MessageID
	session.ExpiresAt = time.Now().UTC().Add(addShowSessionLifetime)

	switch parts[2] {
	case "select":
		if len(parts) != 4 {
			return
		}
		index, err := strconv.Atoi(parts[3])
		if err != nil || index < 0 || index >= len(session.Shows) {
			return
		}
		session.Selected = session.Shows[index]
		bot.editRemoval(ctx, session, removalConfirmation(session.Selected), session.confirmationKeyboard())
	case "page":
		if len(parts) != 4 {
			return
		}
		page, err := strconv.Atoi(parts[3])
		if err != nil || page < 0 || page >= session.pageCount() {
			return
		}
		session.Page = page
		session.Selected = nil
		bot.editRemoval(ctx, session, session.listText(), session.listKeyboard())
	case "back":
		session.Selected = nil
		bot.editRemoval(ctx, session, session.listText(), session.listKeyboard())
	case "confirm":
		bot.confirmRemoveShow(ctx, session)
	case "cancel":
		delete(bot.removals, callback.From.ID)
		bot.editRemoval(ctx, session, "Show removal cancelled.", InlineKeyboardMarkup{})
	}
}

func (bot *Bot) confirmRemoveShow(ctx context.Context, session *removeShowSession) {
	if session.Selected == nil {
		return
	}
	show, err := common.RemoveShow(session.Selected.ID)
	if err != nil {
		message := "Could not remove show: " + err.Error()
		markup := keyboard(row(button("Try again", session.callback("confirm")), button("Cancel", session.callback("cancel"))))
		bot.editRemoval(ctx, session, message, markup)
		return
	}
	delete(bot.removals, session.UserID)
	message := fmt.Sprintf("%s removed from Couchness tracking.\n\nMedia files and Transmission torrents were not changed.", show.Title)
	bot.editRemoval(ctx, session, message, InlineKeyboardMarkup{})
}

func removalConfirmation(show *models.Show) string {
	return fmt.Sprintf("Stop tracking %s?\n\nThis removes only its Couchness record. Local media files and Transmission torrents remain untouched.", show.Title)
}

func (bot *Bot) editRemoval(ctx context.Context, session *removeShowSession, message string, markup InlineKeyboardMarkup) {
	_ = bot.client.editMessage(ctx, session.ChatID, session.MessageID, message, markup)
}

func (session *removeShowSession) callback(action string) string {
	return "remove|" + session.ID + "|" + action
}

func (session *removeShowSession) pageCount() int {
	return (len(session.Shows) + removeShowPageSize - 1) / removeShowPageSize
}

func (session *removeShowSession) listText() string {
	return fmt.Sprintf("Choose a show to stop tracking.\nPage %d of %d", session.Page+1, session.pageCount())
}

func (session *removeShowSession) listKeyboard() InlineKeyboardMarkup {
	start := session.Page * removeShowPageSize
	end := start + removeShowPageSize
	if end > len(session.Shows) {
		end = len(session.Shows)
	}
	rows := make([][]InlineKeyboardButton, 0, removeShowPageSize+2)
	for index := start; index < end; index++ {
		show := session.Shows[index]
		label := truncateButtonText(show.Title+" ("+show.ID+")", 60)
		rows = append(rows, row(button(label, session.callback("select|"+strconv.Itoa(index)))))
	}
	navigation := []InlineKeyboardButton{}
	if session.Page > 0 {
		navigation = append(navigation, button("Previous", session.callback("page|"+strconv.Itoa(session.Page-1))))
	}
	if session.Page+1 < session.pageCount() {
		navigation = append(navigation, button("Next", session.callback("page|"+strconv.Itoa(session.Page+1))))
	}
	if len(navigation) > 0 {
		rows = append(rows, navigation)
	}
	rows = append(rows, row(button("Cancel", session.callback("cancel"))))
	return keyboard(rows...)
}

func (session *removeShowSession) confirmationKeyboard() InlineKeyboardMarkup {
	return keyboard(
		row(button("Remove tracking", session.callback("confirm"))),
		row(button("Back", session.callback("back")), button("Cancel", session.callback("cancel"))),
	)
}

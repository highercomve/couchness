package telegram

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/highercomve/couchness/common"
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

const inviteLifetime = 10 * time.Minute

// Bot exposes Couchness operations through Telegram commands.
type Bot struct {
	client              *apiClient
	username            string
	sessions            map[int64]*addShowSession
	movieSessions       map[int64]*addMovieSession
	removals            map[int64]*removeShowSession
	searchShows         func(string, string) (*common.OmdbResponse, error)
	searchMovieTorrents func(*models.Movie) (models.Episodes, error)
	downloadMovie       func(*models.Movie, *models.TorrentInfo) error
}

// New validates token and creates a Telegram bot.
func New(ctx context.Context, token string) (*Bot, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("COUCHNESS_TELEGRAM_BOT_TOKEN is required")
	}
	client := newAPIClient(token)
	identity, err := client.getMe(ctx)
	if err != nil {
		return nil, err
	}
	return &Bot{
		client:              client,
		username:            identity.Username,
		sessions:            make(map[int64]*addShowSession),
		movieSessions:       make(map[int64]*addMovieSession),
		removals:            make(map[int64]*removeShowSession),
		searchShows:         common.SearchShowInfo,
		searchMovieTorrents: common.SearchMovieTorrents,
		downloadMovie:       common.DownloadMovie,
	}, nil
}

// Username returns bot's Telegram username.
func (bot *Bot) Username() string {
	return bot.username
}

// Run receives Telegram updates until context cancellation.
func (bot *Bot) Run(ctx context.Context) error {
	if err := bot.client.deleteWebhook(ctx); err != nil {
		return fmt.Errorf("could not prepare Telegram long polling: %w", err)
	}
	if err := bot.client.setCommands(ctx, telegramCommands()); err != nil {
		return fmt.Errorf("could not configure Telegram command menu: %w", err)
	}
	var offset int64
	for {
		updates, err := bot.client.getUpdates(ctx, offset)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(3 * time.Second):
				continue
			}
		}

		for _, update := range updates {
			offset = update.UpdateID + 1
			if update.Message != nil {
				bot.handleMessage(ctx, update.Message)
			}
			if update.CallbackQuery != nil {
				bot.handleCallback(ctx, update.CallbackQuery)
			}
		}
	}
}

func (bot *Bot) handleMessage(ctx context.Context, message *Message) {
	if message.Chat.Type != "private" || strings.TrimSpace(message.Text) == "" {
		return
	}

	configuration, err := storage.GetTelegramConfiguration()
	if err != nil {
		bot.reply(ctx, message.Chat.ID, "Could not load Telegram configuration: "+err.Error())
		return
	}
	if !configuration.Enabled {
		bot.reply(ctx, message.Chat.ID, "Telegram integration is disabled.")
		return
	}

	isCommand := strings.HasPrefix(message.Text, "/")
	var command string
	var arguments []string
	if isCommand {
		fields := strings.Fields(message.Text)
		command = strings.ToLower(strings.Split(fields[0], "@")[0])
		arguments = fields[1:]
	}

	user := configuration.Users[strconv.FormatInt(message.From.ID, 10)]
	if user == nil {
		if command == "/start" && len(arguments) == 1 {
			if bot.claimInvite(configuration, arguments[0], message.From.ID) {
				bot.reply(ctx, message.Chat.ID, "Access granted. Use /help to see commands.")
				return
			}
		}
		bot.reply(ctx, message.Chat.ID, fmt.Sprintf("Not authorized. Your Telegram user ID is %d. Ask an owner to add you or send an invite link.", message.From.ID))
		return
	}
	if !isCommand {
		bot.handleSessionText(ctx, message, user)
		return
	}

	switch command {
	case "/start", "/help":
		bot.reply(ctx, message.Chat.ID, helpText(user.Role))
	case "/shows":
		bot.listShows(ctx, message.Chat.ID)
	case "/movies":
		bot.listMovies(ctx, message.Chat.ID)
	case "/show":
		bot.show(ctx, message.Chat.ID, arguments)
	case "/update":
		bot.runShowAction(ctx, message.Chat.ID, user, arguments, "update")
	case "/download":
		bot.runShowAction(ctx, message.Chat.ID, user, arguments, "download")
	case "/update_all":
		bot.updateAll(ctx, message.Chat.ID, user)
	case "/add_show":
		bot.startAddShow(ctx, message, user, arguments)
	case "/add_movie":
		bot.startAddMovie(ctx, message, user, arguments)
	case "/remove_show":
		bot.startRemoveShow(ctx, message, user, arguments)
	case "/cancel":
		bot.cancelSession(ctx, message.Chat.ID, message.From.ID)
	case "/users":
		bot.listUsers(ctx, message.Chat.ID, user, configuration)
	case "/invite":
		bot.createInvite(ctx, message.Chat.ID, user, configuration, arguments)
	case "/revoke":
		bot.revokeUser(ctx, message.Chat.ID, user, configuration, arguments)
	default:
		bot.reply(ctx, message.Chat.ID, "Unknown command. Use /help.")
	}
}

func (bot *Bot) listShows(ctx context.Context, chatID int64) {
	shows, err := common.GetShows()
	if err != nil {
		bot.reply(ctx, chatID, "Could not list shows: "+err.Error())
		return
	}
	if len(shows) == 0 {
		bot.reply(ctx, chatID, "No shows found.")
		return
	}
	lines := make([]string, 0, len(shows))
	for _, show := range shows {
		lines = append(lines, show.Summary())
	}
	bot.reply(ctx, chatID, strings.Join(lines, "\n"))
}

func (bot *Bot) listMovies(ctx context.Context, chatID int64) {
	movies, err := common.GetMovies()
	if err != nil {
		bot.reply(ctx, chatID, "Could not list movies: "+err.Error())
		return
	}
	if len(movies) == 0 {
		bot.reply(ctx, chatID, "No movies found.")
		return
	}
	lines := make([]string, 0, len(movies))
	for _, movie := range movies {
		lines = append(lines, movie.Summary())
	}
	bot.reply(ctx, chatID, strings.Join(lines, "\n"))
}

func (bot *Bot) show(ctx context.Context, chatID int64, arguments []string) {
	if len(arguments) != 1 {
		bot.reply(ctx, chatID, "Usage: /show <show_id>")
		return
	}
	value, err := common.GetShow(arguments[0], false)
	if err != nil {
		bot.reply(ctx, chatID, "Could not load show: "+err.Error())
		return
	}
	show := value.(*models.Show)
	bot.reply(ctx, chatID, show.Summary()+"\nDirectory: "+show.Directory)
}

func (bot *Bot) runShowAction(ctx context.Context, chatID int64, user *models.TelegramUser, arguments []string, action string) {
	if user.Role == models.TelegramRoleViewer {
		bot.reply(ctx, chatID, "Permission denied.")
		return
	}
	if len(arguments) != 1 {
		bot.reply(ctx, chatID, fmt.Sprintf("Usage: /%s <show_id>", action))
		return
	}

	bot.reply(ctx, chatID, fmt.Sprintf("Starting %s for %s.", action, arguments[0]))
	var err error
	if action == "update" {
		err = common.Update(arguments[0])
	} else {
		err = common.Download(arguments[0])
	}
	if err != nil {
		bot.reply(ctx, chatID, fmt.Sprintf("%s failed: %s", action, err))
		return
	}
	bot.reply(ctx, chatID, fmt.Sprintf("%s completed for %s.", strings.Title(action), arguments[0]))
}

func (bot *Bot) updateAll(ctx context.Context, chatID int64, user *models.TelegramUser) {
	if user.Role == models.TelegramRoleViewer {
		bot.reply(ctx, chatID, "Permission denied.")
		return
	}
	bot.reply(ctx, chatID, "Updating all shows.")
	messages := []string{}
	err := common.UpdateAll(func(message string) {
		messages = append(messages, message)
	})
	if err != nil {
		bot.reply(ctx, chatID, "Update failed: "+err.Error())
		return
	}
	if len(messages) > 0 {
		bot.reply(ctx, chatID, strings.Join(messages, "\n"))
	}
	bot.reply(ctx, chatID, "Update completed.")
}

func (bot *Bot) listUsers(ctx context.Context, chatID int64, actor *models.TelegramUser, configuration *models.TelegramConfiguration) {
	if !isAdministrator(actor.Role) {
		bot.reply(ctx, chatID, "Permission denied.")
		return
	}
	lines := []string{}
	for _, user := range configuration.Users {
		lines = append(lines, fmt.Sprintf("%d: %s", user.ID, user.Role))
	}
	if len(lines) == 0 {
		lines = append(lines, "No authorized users.")
	}
	bot.reply(ctx, chatID, strings.Join(lines, "\n"))
}

func (bot *Bot) createInvite(ctx context.Context, chatID int64, actor *models.TelegramUser, configuration *models.TelegramConfiguration, arguments []string) {
	if !isAdministrator(actor.Role) {
		bot.reply(ctx, chatID, "Permission denied.")
		return
	}
	role := models.TelegramRoleUser
	if len(arguments) > 0 {
		role = strings.ToLower(arguments[0])
	}
	if !models.ValidTelegramRole(role) || role == models.TelegramRoleOwner || (role == models.TelegramRoleAdmin && actor.Role != models.TelegramRoleOwner) {
		bot.reply(ctx, chatID, "Invalid role. Owners may invite admin, user, or viewer. Admins may invite user or viewer.")
		return
	}
	code, err := randomCode()
	if err != nil {
		bot.reply(ctx, chatID, "Could not create invite: "+err.Error())
		return
	}
	configuration.Invites[code] = &models.TelegramInvite{
		Code:      code,
		Role:      role,
		CreatedBy: actor.ID,
		ExpiresAt: time.Now().UTC().Add(inviteLifetime),
	}
	if err := storage.SaveTelegramConfiguration(configuration); err != nil {
		bot.reply(ctx, chatID, "Could not save invite: "+err.Error())
		return
	}
	bot.reply(ctx, chatID, fmt.Sprintf("Single-use %s invite, valid for 10 minutes:\nhttps://t.me/%s?start=%s", role, bot.username, code))
}

func (bot *Bot) claimInvite(configuration *models.TelegramConfiguration, code string, userID int64) bool {
	now := time.Now().UTC()
	for key, invite := range configuration.Invites {
		if now.After(invite.ExpiresAt) {
			delete(configuration.Invites, key)
		}
	}
	invite := configuration.Invites[code]
	if invite == nil || now.After(invite.ExpiresAt) {
		return false
	}
	configuration.Users[strconv.FormatInt(userID, 10)] = &models.TelegramUser{ID: userID, Role: invite.Role, AddedAt: now}
	delete(configuration.Invites, code)
	return storage.SaveTelegramConfiguration(configuration) == nil
}

func (bot *Bot) revokeUser(ctx context.Context, chatID int64, actor *models.TelegramUser, configuration *models.TelegramConfiguration, arguments []string) {
	if !isAdministrator(actor.Role) || len(arguments) != 1 {
		bot.reply(ctx, chatID, "Usage for administrators: /revoke <telegram_user_id>")
		return
	}
	targetID, err := strconv.ParseInt(arguments[0], 10, 64)
	if err != nil {
		bot.reply(ctx, chatID, "Telegram user ID must be a number.")
		return
	}
	target := configuration.Users[strconv.FormatInt(targetID, 10)]
	if target == nil {
		bot.reply(ctx, chatID, "User not found.")
		return
	}
	if target.Role == models.TelegramRoleOwner || (target.Role == models.TelegramRoleAdmin && actor.Role != models.TelegramRoleOwner) {
		bot.reply(ctx, chatID, "Only owner may revoke administrators; owner accounts must be managed through CLI.")
		return
	}
	delete(configuration.Users, strconv.FormatInt(targetID, 10))
	if err := storage.SaveTelegramConfiguration(configuration); err != nil {
		bot.reply(ctx, chatID, "Could not revoke user: "+err.Error())
		return
	}
	bot.reply(ctx, chatID, fmt.Sprintf("Revoked user %d.", targetID))
}

func (bot *Bot) reply(ctx context.Context, chatID int64, message string) {
	_ = bot.client.sendMessage(ctx, chatID, message)
}

func isAdministrator(role string) bool {
	return role == models.TelegramRoleOwner || role == models.TelegramRoleAdmin
}

func randomCode() (string, error) {
	value := make([]byte, 18)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func helpText(role string) string {
	commands := []string{
		"/shows - list shows",
		"/movies - list movies",
		"/show <show_id> - show details",
	}
	if role != models.TelegramRoleViewer {
		commands = append(commands,
			"/update <show_id> - scan and update one show",
			"/download <show_id> - queue latest episode",
			"/update_all - scan and update all shows",
		)
	}
	if isAdministrator(role) {
		commands = append(commands,
			"/add_show <title> - add a show with guided setup",
			"/add_movie <title> - find and download a movie",
			"/remove_show - stop tracking a show",
			"/users - list authorized users",
			"/invite [user|viewer|admin] - create invite",
			"/revoke <telegram_user_id> - revoke access",
		)
	}
	return strings.Join(commands, "\n")
}

func telegramCommands() []BotCommand {
	return []BotCommand{
		{Command: "start", Description: "Show available commands"},
		{Command: "shows", Description: "List shows in your library"},
		{Command: "movies", Description: "List movies in your library"},
		{Command: "add_show", Description: "Add a show with guided setup"},
		{Command: "add_movie", Description: "Find and download a movie"},
		{Command: "remove_show", Description: "Stop tracking a show"},
		{Command: "update", Description: "Update one show"},
		{Command: "download", Description: "Download latest episode"},
		{Command: "update_all", Description: "Update all followed shows"},
		{Command: "users", Description: "List authorized users"},
		{Command: "invite", Description: "Invite another user"},
		{Command: "cancel", Description: "Cancel active setup"},
		{Command: "help", Description: "Show help"},
	}
}

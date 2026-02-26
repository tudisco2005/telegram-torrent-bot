package commands

import (
	"github.com/pyed/transmission"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	"github.com/tudisco2005/telegram-torrent-bot/commands/helpers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Checking lists torrents that are checking/verifying
func Checking(h *handlers.Handler, ud tgbotapi.Update, cmd string) {
	helpers.SendTorrentsByStatus(
		h,
		ud,
		cmd,
		"checking",
		"No torrents verifying",
		[]interface{}{"plain"},
		transmission.StatusChecking,
		transmission.StatusCheckPending,
	)
}

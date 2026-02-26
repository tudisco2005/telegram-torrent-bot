package commands

import (
	"github.com/pyed/transmission"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	"github.com/tudisco2005/telegram-torrent-bot/commands/helpers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Paused lists torrents that are paused
func Paused(h *handlers.Handler, ud tgbotapi.Update, cmd string) {
	helpers.SendTorrentsByStatus(
		h,
		ud,
		cmd,
		"paused",
		"No paused torrents",
		nil,
		transmission.StatusStopped,
	)
}

package commands

import (
	"github.com/pyed/transmission"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	"github.com/tudisco2005/telegram-torrent-bot/commands/helpers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Downs lists torrents that are downloading
func Downs(h *handlers.Handler, ud tgbotapi.Update, cmd string) {
	helpers.SendTorrentsByStatus(
		h,
		ud,
		cmd,
		"downs",
		"No downloads",
		nil,
		transmission.StatusDownloading,
		transmission.StatusDownloadPending,
	)
}

package commands

import (
	"github.com/pyed/transmission"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	"github.com/tudisco2005/telegram-torrent-bot/commands/helpers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Seeding lists torrents that are seeding
func Seeding(h *handlers.Handler, ud tgbotapi.Update, cmd string) {
	helpers.SendTorrentsByStatus(
		h,
		ud,
		cmd,
		"seeding",
		"No torrents seeding",
		[]interface{}{"plain"},
		transmission.StatusSeeding,
		transmission.StatusSeedPending,
	)
}

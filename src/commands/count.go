package commands

import (
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Count shows the number of torrents per status
func Count(h *handlers.Handler, ud tgbotapi.Update, cmd string) {
	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*count:* "+err.Error(), cmd)
		return
	}

	var downloading, seeding, stopped, checking, downloadingQ, seedingQ, checkingQ int

	for i := range torrents {
		switch torrents[i].Status {
		case 3:
			downloading++
		case 6:
			seeding++
		case 0:
			stopped++
		case 4:
			checking++
		case 2:
			downloadingQ++
		case 5:
			seedingQ++
		case 1:
			checkingQ++
		}
	}

	msg := h.FormatOutputString(cmd,
		downloading, seeding, stopped, checking, downloadingQ, seedingQ, checkingQ, len(torrents))

	h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
}

package commands

import (
	"bytes"

	"github.com/pyed/transmission"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Paused lists torrents that are paused
func Paused(h *handlers.Handler, ud tgbotapi.Update, cmd string) {
	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*paused:* "+err.Error(), cmd)
		return
	}

	buf := new(bytes.Buffer)
	for i := range torrents {
		if torrents[i].Status == transmission.StatusStopped {
			buf.WriteString(h.FormatOutputString(cmd, torrents[i].ID, torrents[i].Name))
		}
	}

	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "No paused torrents", cmd)
		return
	}

	h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)
}

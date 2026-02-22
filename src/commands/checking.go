package commands

import (
	"bytes"

	"github.com/pyed/transmission"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Checking lists torrents that are checking/verifying
func Checking(h *handlers.Handler, ud tgbotapi.Update, cmd string) {
	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*checking:* "+err.Error(), cmd)
		return
	}

	buf := new(bytes.Buffer)
	for i := range torrents {
		if torrents[i].Status == transmission.StatusChecking ||
			torrents[i].Status == transmission.StatusCheckPending {
			buf.WriteString(h.FormatOutputString(cmd, torrents[i].ID, torrents[i].Name))
		}
	}

	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "No torrents verifying", cmd, "plain")
		return
	}

	h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)
}

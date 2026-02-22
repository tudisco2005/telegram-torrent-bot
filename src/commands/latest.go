package commands

import (
	"bytes"
	"strconv"

	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Latest lists the latest n added torrents
func Latest(h *handlers.Handler, ud tgbotapi.Update, tokens []string, cmd string) {
	var (
		n   = 5 // default to 5
		err error
	)

	if len(tokens) > 0 {
		n, err = strconv.Atoi(tokens[0])
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*latest:* "+err.Error(), cmd)
			return
		}
	}

	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*latest:* "+err.Error(), cmd)
		return
	}

	if n <= 0 || n > len(torrents) {
		n = len(torrents)
	}

	torrents.SortAge(true)

	buf := new(bytes.Buffer)
	for i := range torrents[:n] {
		buf.WriteString(h.FormatOutputString(cmd, torrents[i].ID, torrents[i].Name))
	}
	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*latest:* No torrents", cmd, "markdown")
		return
	}
	h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)
}

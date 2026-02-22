package commands

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Search searches for torrents by name
func Search(h *handlers.Handler, ud tgbotapi.Update, tokens []string, cmd string) {

	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*search:* needs an argument", cmd)
		return
	}

	query := strings.Join(tokens, " ")

	regx, err := regexp.Compile("(?i)" + query)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*search:* "+err.Error(), cmd)
		return
	}

	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*search:* "+err.Error(), cmd)
		return
	}

	buf := new(bytes.Buffer)
	for i := range torrents {
		if regx.MatchString(torrents[i].Name) {
			buf.WriteString(h.FormatOutputString(cmd, torrents[i].ID, torrents[i].Name))
		}
	}
	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "No matches!", cmd)
		return
	}
	h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)
}

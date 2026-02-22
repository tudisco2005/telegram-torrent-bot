package commands

import (
	"strconv"

	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Check verifies one or more torrents
func Check(h *handlers.Handler, ud tgbotapi.Update, tokens []string, cmd string) {

	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*check:* needs an argument", cmd)
		return
	}

	if tokens[0] == "all" {

		torrents, err := h.Client.GetTorrents()
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*check:* "+err.Error(), cmd)
			return
		}
		for _, t := range torrents {
			h.Client.VerifyTorrent(t.ID)
		}
		h.SendWithFormat(ud.Message.Chat.ID, "Verifying all torrents", cmd)
		return
	}

	for _, id := range tokens {
		num, err := strconv.Atoi(id)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*check:* "+err.Error(), cmd)
			continue
		}
		status, err := h.Client.VerifyTorrent(num)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*check:* "+err.Error(), cmd)
			continue
		}

		torrent, err := h.Client.GetTorrent(num)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*check:* "+err.Error(), cmd)
			continue
		}
		msg := h.FormatOutputString(cmd, status, torrent.Name)
		h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
	}
}

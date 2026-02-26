package commands

import (
	"strconv"

	"github.com/tudisco2005/telegram-torrent-bot/commands/helpers"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Stop stops one or more torrents
func Stop(h *handlers.Handler, ud tgbotapi.Update, tokens []string, cmd string) {

	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*stop:* needs an argument", cmd)
		return
	}

	if tokens[0] == "all" {

		torrents, err := h.Client.GetTorrents()
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*stop:* "+err.Error(), cmd)
			return
		}
		for _, t := range torrents {
			h.Client.StopTorrent(t.ID)
		}
		helpers.ClearTrackedIDs(h)
		h.SendWithFormat(ud.Message.Chat.ID, "Stopped all torrents", cmd)
		return
	}

	for _, id := range tokens {
		num, err := strconv.Atoi(id)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*stop:* "+err.Error(), cmd)
			continue
		}
		status, err := h.Client.StopTorrent(num)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*stop:* "+err.Error(), cmd)
			continue
		}

		torrent, err := h.Client.GetTorrent(num)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*stop:* "+err.Error(), cmd)
			continue
		}
		msg := h.FormatOutputString(cmd, status, torrent.Name)
		h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)

		if torrent.PercentDone < 1.0 {
			helpers.RemoveTrackedIDs(h, []int{num})
		}
	}
}

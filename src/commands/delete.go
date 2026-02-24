package commands

import (
	"strconv"

	"github.com/tudisco2005/telegram-torrent-bot/commands/helpers"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	"github.com/tudisco2005/telegram-torrent-bot/utils"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Delete deletes one or more torrents (keeps data)
func Delete(h *handlers.Handler, ud tgbotapi.Update, tokens []string, cmd string) {

	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*deldata:* needs an ID", cmd)
		return
	}

	for _, id := range tokens {
		num, err := strconv.Atoi(id)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*deldata:* "+err.Error(), cmd)
			continue
		}

		wasIncomplete := false
		if t, terr := h.Client.GetTorrent(num); terr == nil {
			wasIncomplete = t.PercentDone < 1.0
		} else {
			h.Logger.Printf("[DEBUG] del: failed to inspect torrent %d before delete: %v", num, terr)
		}

		name, err := h.Client.DeleteTorrent(num, false)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*deldata:* "+err.Error(), cmd)
			continue
		}

		if wasIncomplete {
			helpers.RemoveTrackedIDs(h, []int{num})
		}

		safeName := utils.EscapeFileMD(name)
		msg := h.FormatOutputString(cmd, safeName)
		h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
	}
}

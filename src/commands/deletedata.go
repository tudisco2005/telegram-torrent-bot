package commands

import (
	"strconv"

	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	"github.com/tudisco2005/telegram-torrent-bot/utils"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// DeleteData deletes one or more torrents with their data
func DeleteData(h *handlers.Handler, ud tgbotapi.Update, tokens []string, cmd string) {

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

		name, err := h.Client.DeleteTorrent(num, true)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*deldata:* "+err.Error(), cmd)
			continue
		}

		if tracked, terr := utils.LoadTracked(h.CompletedFilePath); terr == nil {
			newTracked := make([]int, 0, len(tracked))
			for _, id := range tracked {
				if id != num {
					newTracked = append(newTracked, id)
				}
			}
			if len(newTracked) != len(tracked) {
				if serr := utils.SaveTracked(h.CompletedFilePath, newTracked); serr != nil {
					h.Logger.Printf("[WARNING] failed to update %s: %v", h.CompletedFilePath, serr)
				}
			}
		} else {
			h.Logger.Printf("[DEBUG] failed to load telegram/completed.json: %v", terr)
		}

		safeName := utils.EscapeFileMD(name)
		msg := h.FormatOutputString(cmd, safeName)
		h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
	}
}

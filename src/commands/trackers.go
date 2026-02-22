package commands

import (
	"bytes"

	tgbotapi "gopkg.in/telegram-bot-api.v4"

	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	"github.com/tudisco2005/telegram-torrent-bot/utils"
)

// Trackers lists all trackers and their torrent count
func Trackers(h *handlers.Handler, ud tgbotapi.Update, cmd string) {
	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*trackers:* "+err.Error(), cmd)
		return
	}

	trackers := make(map[string]int)

	for i := range torrents {
		for _, tracker := range torrents[i].Trackers {
			if matches := utils.TrackerRegex.FindStringSubmatch(tracker.Announce); matches != nil {
				trackers[matches[1]]++
			}
		}
	}

	buf := new(bytes.Buffer)
	for k, v := range trackers {
		buf.WriteString(h.FormatOutputString(cmd, v, k))
	}

	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "No trackers!", cmd)
		return
	}
	h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)
}

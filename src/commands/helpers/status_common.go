package helpers

import (
	"bytes"

	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	"github.com/tudisco2005/telegram-torrent-bot/utils"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// sendTorrentsByStatus fetches torrents once, filters by statuses, and sends
// a formatted list. It centralizes the repeated logic used by status commands
// like downs/paused/checking/seeding.
func SendTorrentsByStatus(
	h *handlers.Handler,
	ud tgbotapi.Update,
	cmd string,
	errorPrefix string,
	emptyMessage string,
	emptyFormats []interface{},
	statuses ...int,
) {
	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*"+errorPrefix+":* "+err.Error(), cmd)
		return
	}

	allowed := make(map[int]struct{}, len(statuses))
	for _, status := range statuses {
		allowed[status] = struct{}{}
	}

	buf := new(bytes.Buffer)
	for i := range torrents {
		if _, ok := allowed[torrents[i].Status]; ok {
			buf.WriteString(h.FormatOutputString(cmd, torrents[i].ID, utils.EscapeFileMD(torrents[i].Name)))
		}
	}

	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, emptyMessage, cmd, emptyFormats...)
		return
	}

	h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)
}

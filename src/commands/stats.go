package commands

import (
	"github.com/dustin/go-humanize"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Stats shows transmission statistics
func Stats(h *handlers.Handler, ud tgbotapi.Update, cmd string) {
	stats, err := h.Client.GetStats()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*stats:* "+err.Error(), cmd)
		return
	}

	msg := h.FormatOutputString(cmd,
		stats.TorrentCount,
		stats.ActiveTorrentCount,
		stats.PausedTorrentCount,
		humanize.Bytes(stats.CurrentStats.DownloadedBytes),
		humanize.Bytes(stats.CurrentStats.UploadedBytes),
		stats.CurrentActiveTime(),
		stats.CumulativeStats.SessionCount,
		humanize.Bytes(stats.CumulativeStats.DownloadedBytes),
		humanize.Bytes(stats.CumulativeStats.UploadedBytes),
		stats.CumulativeActiveTime(),
	)

	h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
}

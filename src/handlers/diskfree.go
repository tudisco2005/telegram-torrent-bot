package handlers

import (
	"syscall"

	"github.com/dustin/go-humanize"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// DiskFree reports used disk space for the default download location
func (h *Handler) DiskUsage(ud tgbotapi.Update, cmd string) {
	// Choose target path: prefer DefaultDownloadLocation, fallback to root
	path := h.DefaultDownloadLocation
	if path == "" {
		path = "/"
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*diskusage:* "+err.Error(), cmd)
		return
	}

	// Available blocks * size per block = available bytes
	avail := uint64(stat.Bavail) * uint64(stat.Bsize)
	total := uint64(stat.Blocks) * uint64(stat.Bsize)
	used := total - avail
	var pct int
	if total > 0 {
		pct = int((used * 100) / total)
	} else {
		pct = 0
	}

	usedStr := humanize.Bytes(used)
	totalStr := humanize.Bytes(total)
	// The FormatOutputString expects command -> format template; use that (used, total, pct)
	out := h.FormatOutputString(cmd, usedStr, totalStr, pct)
	h.SendWithFormat(ud.Message.Chat.ID, out, cmd)
}

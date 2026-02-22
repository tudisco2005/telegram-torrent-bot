package commands

import (
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/dustin/go-humanize"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// DiskFree reports used disk space for the default download location
func DiskUsage(h *handlers.Handler, ud tgbotapi.Update, cmd string) {

	dlPath := h.DefaultDownloadLocation
	if dlPath == "" {
		dlPath = "/"
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(dlPath, &stat); err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*diskusage:* "+err.Error(), cmd)
		return
	}

	avail := uint64(stat.Bavail) * uint64(stat.Bsize)
	total := uint64(stat.Blocks) * uint64(stat.Bsize)
	used := total - avail
	var pct int
	if total > 0 {
		pct = int((used * 100) / total)
	} else {
		pct = 0
	}
	dlUsed := humanize.Bytes(used)
	dlTotal := humanize.Bytes(total)

	mvPath := h.DefaultMoveLocation
	if mvPath == "" {
		mvPath = os.Getenv("DEFAULT_MOVE_LOCATION")
	}
	mvStatus := "not configured"
	if mvPath != "" {
		var mstat syscall.Statfs_t
		if err := syscall.Statfs(mvPath, &mstat); err == nil {
			mavail := uint64(mstat.Bavail) * uint64(mstat.Bsize)
			mtotal := uint64(mstat.Blocks) * uint64(mstat.Bsize)
			mused := mtotal - mavail
			mpct := 0
			if mtotal > 0 {
				mpct = int((mused * 100) / mtotal)
			}
			mvStatus = humanize.Bytes(mused) + " / " + humanize.Bytes(mtotal) + " (" + strconv.Itoa(mpct) + "%)"
		} else {
			mvStatus = "error: " + err.Error()
		}
	}

	out := fmt.Sprintf("Download `%s`: %s / %s (%d%%)\nMove `%s`: %s", dlPath, dlUsed, dlTotal, pct, mvPath, mvStatus)
	h.SendWithFormat(ud.Message.Chat.ID, out, cmd)
}

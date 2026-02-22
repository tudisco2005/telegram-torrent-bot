package commands

import (
	"fmt"
	"strings"
	"syscall"
	"time"

	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	"github.com/tudisco2005/telegram-torrent-bot/utils"
)

func withSourceSuffix(message string, source string) string {
	if strings.TrimSpace(source) == "" {
		return message
	}
	return message + " — " + source
}

// validateAddedTorrent checks disk-space/error constraints after adding a torrent.
// It returns metadataPending=true when metadata polling goroutine was started,
// and blocked=true when the torrent has been stopped/deleted due to validation failures.
func validateAddedTorrent(h *handlers.Handler, chatID int64, cmd string, torrentID int, source string, metadataPendingMessage string) (metadataPending bool, blocked bool) {
	path := h.DefaultDownloadLocation
	if path == "" {
		path = "/"
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		h.Logger.Printf("[WARNING] validateAddedTorrent: statfs failed on %s: %v", path, err)
		return false, false
	}

	avail := uint64(stat.Bavail) * uint64(stat.Bsize)
	torrentFull, err := h.Client.GetTorrent(torrentID)
	if err != nil {
		h.Logger.Printf("[WARNING] validateAddedTorrent: failed to load torrent id=%d: %v", torrentID, err)
		return false, false
	}

	if torrentFull.SizeWhenDone == 0 {
		h.SendWithFormat(chatID, metadataPendingMessage, cmd)
		go watchMetadataAndSpace(h, chatID, cmd, torrentID, source, path)
		return true, false
	}

	if torrentFull.SizeWhenDone > 0 && avail < uint64(torrentFull.SizeWhenDone) {
		if _, err := h.Client.StopTorrent(torrentID); err != nil {
			h.Logger.Printf("[WARNING] validateAddedTorrent: failed to stop torrent id=%d: %v", torrentID, err)
		}
		h.SendWithFormat(chatID, "Not enough space left, torrent stopped", cmd)
		return false, true
	}

	if torrentFull.Error != 0 || strings.Contains(strings.ToLower(torrentFull.ErrorString), "no space") {
		if _, derr := h.Client.DeleteTorrent(torrentID, false); derr != nil {
			h.Logger.Printf("[WARNING] failed to delete torrent id=%d after error: %v", torrentID, derr)
		}
		errMsg := torrentFull.ErrorString
		if errMsg == "" {
			errMsg = "Not enough space left (transmission reported error)"
		}
		h.SendWithFormat(chatID, withSourceSuffix("*add:* "+errMsg, source), cmd)
		return false, true
	}

	return false, false
}

func watchMetadataAndSpace(h *handlers.Handler, chatID int64, cmd string, torrentID int, source string, path string) {
	for i := 0; i < 60; i++ {
		time.Sleep(5 * time.Second)
		t, err := h.Client.GetTorrent(torrentID)
		if err != nil {
			break
		}

		if t.Error != 0 || strings.Contains(strings.ToLower(t.ErrorString), "no space") {
			if _, derr := h.Client.DeleteTorrent(torrentID, false); derr != nil {
				h.Logger.Printf("[WARNING] failed to delete torrent id=%d after error: %v", torrentID, derr)
			}
			errMsg := t.ErrorString
			if errMsg == "" {
				errMsg = "Not enough space left (transmission reported error)"
			}
			h.SendWithFormat(chatID, withSourceSuffix("*add:* "+errMsg, source), cmd)
			return
		}

		if t.SizeWhenDone > 0 {
			var need uint64
			if t.Have() > 0 {
				need = uint64(t.SizeWhenDone) - uint64(t.Have())
			} else {
				need = uint64(t.SizeWhenDone)
			}
			var st syscall.Statfs_t
			if err := syscall.Statfs(path, &st); err == nil {
				availNow := uint64(st.Bavail) * uint64(st.Bsize)
				if need > 0 && availNow < need {
					if _, serr := h.Client.StopTorrent(torrentID); serr == nil {
						h.SendWithFormat(chatID, fmt.Sprintf("Not enough space for torrent %s (id=%d); torrent stopped", t.Name, torrentID), cmd)
					}
					return
				}
			}

			h.SendWithFormat(chatID, fmt.Sprintf("*Metadata fetched* for torrent %s (id=%d)", utils.EscapeFileMD(t.Name), torrentID), cmd)
			return
		}
	}

	h.SendWithFormat(chatID, "Metadata not available after timeout — no action taken", cmd)
}

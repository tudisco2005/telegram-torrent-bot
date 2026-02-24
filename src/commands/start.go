package commands

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"

	"github.com/tudisco2005/telegram-torrent-bot/commands/helpers"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Start starts one or more torrents
func Start(h *handlers.Handler, ud tgbotapi.Update, tokens []string, cmd string) {

	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*start:* needs an argument", cmd)
		return
	}

	if tokens[0] == "all" {

		torrents, err := h.Client.GetTorrents()
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*start:* "+err.Error(), cmd)
			return
		}

		path := h.DefaultDownloadLocation
		if path == "" {
			path = "/"
		}
		var stat syscall.Statfs_t
		var avail uint64
		if err := syscall.Statfs(path, &stat); err == nil {
			avail = uint64(stat.Bavail) * uint64(stat.Bsize)
		}

		started := 0
		startedIncomplete := make([]int, 0)
		for _, t := range torrents {

			if t.SizeWhenDone > 0 {
				var have uint64
				if t.Have() > 0 {
					have = uint64(t.Have())
				}
				var remaining uint64
				if uint64(t.SizeWhenDone) > have {
					remaining = uint64(t.SizeWhenDone) - have
				}
				if remaining > 0 && avail > 0 && avail < remaining {
					h.SendWithFormat(ud.Message.Chat.ID, fmt.Sprintf("Not enough space left for torrent %s (id=%d), not starting", t.Name, t.ID), cmd)
					continue
				}
			}

			if t.Error != 0 || strings.Contains(strings.ToLower(t.ErrorString), "no space") {
				h.SendWithFormat(ud.Message.Chat.ID, fmt.Sprintf("Torrent %s (id=%d) has error: %s", t.Name, t.ID, t.ErrorString), cmd)
				continue
			}

			if _, err := h.Client.StartTorrent(t.ID); err == nil {
				started++
				if t.PercentDone < 1.0 {
					startedIncomplete = append(startedIncomplete, t.ID)
				}
			}
		}
		if len(startedIncomplete) > 0 {
			helpers.AddTrackedIDs(h, startedIncomplete)
		}
		h.SendWithFormat(ud.Message.Chat.ID, fmt.Sprintf("Started %d torrents", started), cmd)
		return
	}

	for _, id := range tokens {
		num, err := strconv.Atoi(id)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*start:* "+err.Error(), cmd)
			continue
		}

		torrent, err := h.Client.GetTorrent(num)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*start:* "+err.Error(), cmd)
			continue
		}

		path := h.DefaultDownloadLocation
		if path == "" {
			path = "/"
		}
		var stat syscall.Statfs_t
		if err := syscall.Statfs(path, &stat); err == nil {
			avail := uint64(stat.Bavail) * uint64(stat.Bsize)
			if torrent.SizeWhenDone > 0 {
				var have uint64
				if torrent.Have() > 0 {
					have = uint64(torrent.Have())
				}
				var remaining uint64
				if uint64(torrent.SizeWhenDone) > have {
					remaining = uint64(torrent.SizeWhenDone) - have
				}
				if remaining > 0 && avail > 0 && avail < remaining {
					h.SendWithFormat(ud.Message.Chat.ID, "Not enough space left, not starting", cmd)
					continue
				}
			}
			if torrent.Error != 0 || strings.Contains(strings.ToLower(torrent.ErrorString), "no space") {
				h.SendWithFormat(ud.Message.Chat.ID, "*start:* "+torrent.ErrorString, cmd)
				continue
			}
		}

		status, err := h.Client.StartTorrent(num)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*start:* "+err.Error(), cmd)
			continue
		}
		msg := h.FormatOutputString(cmd, status, torrent.Name)
		h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)

		if torrent.PercentDone < 1.0 {
			helpers.AddTrackedIDs(h, []int{num})
		}
	}
}

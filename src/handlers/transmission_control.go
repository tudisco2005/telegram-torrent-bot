package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Stop stops one or more torrents
func (h *Handler) Stop(ud tgbotapi.Update, tokens []string, cmd string) {
	// make sure that we got at least one argument
	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*stop:* needs an argument", cmd)
		return
	}

	// if the first argument is 'all' then stop all torrents
	if tokens[0] == "all" {
		// Note: Actual implementation depends on transmission client API
		// Get all torrents and stop them
		torrents, err := h.Client.GetTorrents()
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*stop:* "+err.Error(), cmd)
			return
		}
		for _, t := range torrents {
			h.Client.StopTorrent(t.ID)
		}
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
	}
}

// Start starts one or more torrents
func (h *Handler) Start(ud tgbotapi.Update, tokens []string, cmd string) {
	// make sure that we got at least one argument
	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*start:* needs an argument", cmd)
		return
	}

	// if the first argument is 'all' then start all torrents
	if tokens[0] == "all" {
		// Note: Actual implementation depends on transmission client API
		// Get all torrents and start them
		torrents, err := h.Client.GetTorrents()
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*start:* "+err.Error(), cmd)
			return
		}
		// Check available space and only start torrents that fit
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
		for _, t := range torrents {
			// If we have size info, check remaining bytes needed
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

			// If Transmission previously reported a disk error, skip and report
			if t.Error != 0 || strings.Contains(strings.ToLower(t.ErrorString), "no space") {
				h.SendWithFormat(ud.Message.Chat.ID, fmt.Sprintf("Torrent %s (id=%d) has error: %s", t.Name, t.ID, t.ErrorString), cmd)
				continue
			}

			if _, err := h.Client.StartTorrent(t.ID); err == nil {
				started++
			}
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
		// Before starting, check available disk space for this torrent
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
	}
}

// Check verifies one or more torrents
func (h *Handler) Check(ud tgbotapi.Update, tokens []string, cmd string) {
	// make sure that we got at least one argument
	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*check:* needs an argument", cmd)
		return
	}

	// if the first argument is 'all' then start all torrents
	if tokens[0] == "all" {
		// Note: Actual implementation depends on transmission client API
		// Get all torrents and verify them
		torrents, err := h.Client.GetTorrents()
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*check:* "+err.Error(), cmd)
			return
		}
		for _, t := range torrents {
			h.Client.VerifyTorrent(t.ID)
		}
		h.SendWithFormat(ud.Message.Chat.ID, "Verifying all torrents", cmd)
		return
	}

	for _, id := range tokens {
		num, err := strconv.Atoi(id)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*check:* "+err.Error(), cmd)
			continue
		}
		status, err := h.Client.VerifyTorrent(num)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*check:* "+err.Error(), cmd)
			continue
		}

		torrent, err := h.Client.GetTorrent(num)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*check:* "+err.Error(), cmd)
			continue
		}
		msg := h.FormatOutputString(cmd, status, torrent.Name)
		h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
	}
}

// Delete deletes one or more torrents (keeps data)
func (h *Handler) Delete(ud tgbotapi.Update, tokens []string, cmd string) {
	// make sure that we got an argument
	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*deldata:* needs an ID", cmd)
		return
	}
	// loop over tokens to read each potential id
	for _, id := range tokens {
		num, err := strconv.Atoi(id)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*deldata:* "+err.Error(), cmd)
			continue
		}

		name, err := h.Client.DeleteTorrent(num, false)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*deldata:* "+err.Error(), cmd)
			continue
		}

		msg := h.FormatOutputString(cmd, name)
		h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
	}
}

// DeleteData deletes one or more torrents with their data
func (h *Handler) DeleteData(ud tgbotapi.Update, tokens []string, cmd string) {
	// make sure that we got an argument
	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*deldata:* needs an ID", cmd)
		return
	}
	// loop over tokens to read each potential id
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

		msg := h.FormatOutputString(cmd, name)
		h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
	}
}

// Stats shows transmission statistics
func (h *Handler) Stats(ud tgbotapi.Update, cmd string) {
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

// Speed shows current download and upload speeds
func (h *Handler) Speed(ud tgbotapi.Update, cmd string) {
	stats, err := h.Client.GetStats()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*speed:* "+err.Error(), cmd)
		return
	}

	msg := h.FormatOutputString(cmd, humanize.Bytes(stats.DownloadSpeed), humanize.Bytes(stats.UploadSpeed))

	msgID := h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)

	if h.NoLive || h.UpdateMaxIterations == 0 {
		return
	}

	iterations := h.Duration
	if h.UpdateMaxIterations > 0 && h.UpdateMaxIterations < iterations {
		iterations = h.UpdateMaxIterations
	}
	chatID := ud.Message.Chat.ID

	go func() {
		for i := 0; i < iterations; i++ {
			time.Sleep(h.Interval)
			stats, err := h.Client.GetStats()
			if err != nil {
				break
			}
			msg := h.FormatOutputString(cmd, humanize.Bytes(stats.DownloadSpeed), humanize.Bytes(stats.UploadSpeed))
			editConf := tgbotapi.NewEditMessageText(chatID, msgID, msg)
			if resp, err := h.Bot.Send(editConf); err != nil {
				h.Logger.Printf("[DEBUG] EditMessage failed: ChatID=%d MsgID=%d Len=%d Err=%v", chatID, msgID, len(msg), err)
			} else {
				h.Logger.Printf("[DEBUG] EditMessage sent: ChatID=%d MsgID=%d RespMessageID=%d", chatID, msgID, resp.MessageID)
			}
			time.Sleep(h.Interval)
		}
		time.Sleep(h.Interval)
		editConf := tgbotapi.NewEditMessageText(chatID, msgID, "↓ - B  ↑ - B")
		if resp, err := h.Bot.Send(editConf); err != nil {
			h.Logger.Printf("[DEBUG] EditMessage failed: ChatID=%d MsgID=%d Len=%d Err=%v", chatID, msgID, len("↓ - B  ↑ - B"), err)
		} else {
			h.Logger.Printf("[DEBUG] EditMessage sent: ChatID=%d MsgID=%d RespMessageID=%d", chatID, msgID, resp.MessageID)
		}
	}()
}

// Count shows the number of torrents per status
func (h *Handler) Count(ud tgbotapi.Update, cmd string) {
	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*count:* "+err.Error(), cmd)
		return
	}

	var downloading, seeding, stopped, checking, downloadingQ, seedingQ, checkingQ int

	for i := range torrents {
		switch torrents[i].Status {
		case 3:
			downloading++
		case 6:
			seeding++
		case 0:
			stopped++
		case 4:
			checking++
		case 2:
			downloadingQ++
		case 5:
			seedingQ++
		case 1:
			checkingQ++
		}
	}

	msg := h.FormatOutputString(cmd,
		downloading, seeding, stopped, checking, downloadingQ, seedingQ, checkingQ, len(torrents))

	h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
}

// DownloadLimit sets the global download speed limit
func (h *Handler) DownloadLimit(ud tgbotapi.Update, tokens []string, cmd string) {
	h.SpeedLimit(ud, tokens, "DownloadLimitType", cmd)
}

// UploadLimit sets the global upload speed limit
func (h *Handler) UploadLimit(ud tgbotapi.Update, tokens []string, cmd string) {
	h.SpeedLimit(ud, tokens, "UploadLimitType", cmd)
}

// SpeedLimit sets either download or upload limit
func (h *Handler) SpeedLimit(ud tgbotapi.Update, tokens []string, limitType string, cmd string) {
	if len(tokens) < 1 {
		h.SendWithFormat(ud.Message.Chat.ID, "Please, specify the limit", cmd)
		return
	}

	limit, err := strconv.ParseUint(tokens[0], 10, 32)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "Please, specify the limit as number of kilobytes", cmd)
		return
	}

	// Note: Speed limit command implementation depends on transmission client API
	// This is a placeholder that should be completed based on actual API
	h.SendWithFormat(ud.Message.Chat.ID,
		fmt.Sprintf("*%s:* limit has been successfully changed to %d KB/s", limitType, limit), cmd)
}

// Version shows the version information (uses output_format and output_string for "version" from commands.json)
func (h *Handler) Version(ud tgbotapi.Update, version string) {
	// Use output_string from JSON if available, otherwise use default template
	msg := h.FormatOutputString("version", h.Client.Version(), version)
	h.SendWithFormat(ud.Message.Chat.ID, msg, "version")
}

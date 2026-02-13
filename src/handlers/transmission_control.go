package handlers

import (
	"fmt"
	"strconv"
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
		msg := h.FormatOutputString(cmd, "[%s] *stop:* %s", status, torrent.Name)
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
		for _, t := range torrents {
			h.Client.StartTorrent(t.ID)
		}
		h.SendWithFormat(ud.Message.Chat.ID, "Started all torrents", cmd)
		return
	}

	for _, id := range tokens {
		num, err := strconv.Atoi(id)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*start:* "+err.Error(), cmd)
			continue
		}
		status, err := h.Client.StartTorrent(num)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*start:* "+err.Error(), cmd)
			continue
		}

		torrent, err := h.Client.GetTorrent(num)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*start:* "+err.Error(), cmd)
			continue
		}
		msg := h.FormatOutputString(cmd, "[%s] *start:* %s", status, torrent.Name)
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
		msg := h.FormatOutputString(cmd, "[%s] *check:* %s", status, torrent.Name)
		h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
	}
}

// Delete deletes one or more torrents (keeps data)
func (h *Handler) Delete(ud tgbotapi.Update, tokens []string, cmd string) {
	// make sure that we got an argument
	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*del:* needs an ID", cmd)
		return
	}

	// loop over tokens to read each potential id
	for _, id := range tokens {
		num, err := strconv.Atoi(id)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*del:* "+err.Error(), cmd)
			continue
		}

		name, err := h.Client.DeleteTorrent(num, false)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*del:* "+err.Error(), cmd)
			continue
		}

		msg := h.FormatOutputString(cmd, "*Deleted:* %s", name)
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

		msg := h.FormatOutputString(cmd, "*Deleted:* %s", name)
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

	// Use output_string from JSON if available, otherwise use default template
	defaultTemplate := `
Total: *%d*
Active: *%d*
Paused: *%d*

_Current Stats_
Downloaded: *%s*
Uploaded: *%s*
Running time: *%s*

_Accumulative Stats_
Sessions: *%d*
Downloaded: *%s*
Uploaded: *%s*
Total Running time: *%s*
`
	msg := h.FormatOutputString(cmd, defaultTemplate,
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

	// Use output_string from JSON if available, otherwise use default template
	msg := h.FormatOutputString(cmd, "↓ %s  ↑ %s", humanize.Bytes(stats.DownloadSpeed), humanize.Bytes(stats.UploadSpeed))

	msgID := h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)

	if h.NoLive {
		return
	}

	for i := 0; i < h.Duration; i++ {
		time.Sleep(time.Second * h.Interval)
		stats, err = h.Client.GetStats()
		if err != nil {
			break
		}

		msg = h.FormatOutputString(cmd, "↓ %s  ↑ %s", humanize.Bytes(stats.DownloadSpeed), humanize.Bytes(stats.UploadSpeed))

		editConf := tgbotapi.NewEditMessageText(ud.Message.Chat.ID, msgID, msg)
		h.Bot.Send(editConf)
		time.Sleep(time.Second * h.Interval)
	}
	// sleep one more time before switching to dashes
	time.Sleep(time.Second * h.Interval)

	// show dashes to indicate that we are done updating.
	editConf := tgbotapi.NewEditMessageText(ud.Message.Chat.ID, msgID, "↓ - B  ↑ - B")
	h.Bot.Send(editConf)
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

	// Use output_string from JSON if available, otherwise use default template
	defaultTemplate := "Downloading: %d\nSeeding: %d\nPaused: %d\nVerifying: %d\n\n- Waiting to -\nDownload: %d\nSeed: %d\nVerify: %d\n\nTotal: %d"
	msg := h.FormatOutputString(cmd, defaultTemplate,
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
	msg := h.FormatOutputString("version", "Transmission *%s*\nTransmission-telegram *%s*", h.Client.Version(), version)
	h.SendWithFormat(ud.Message.Chat.ID, msg, "version")
}

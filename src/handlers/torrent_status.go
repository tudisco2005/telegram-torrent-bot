package handlers

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/pyed/transmission"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Downs lists torrents that are downloading
func (h *Handler) Downs(ud tgbotapi.Update, cmd string) {
	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*downs:* "+err.Error(), cmd)
		return
	}

	buf := new(bytes.Buffer)
	for i := range torrents {
		// Downloading or in queue to download
		if torrents[i].Status == transmission.StatusDownloading ||
			torrents[i].Status == transmission.StatusDownloadPending {
			buf.WriteString(h.FormatListLine(cmd, "<%d> %s\n", torrents[i].ID, torrents[i].Name))
		}
	}

	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "No downloads", cmd)
		return
	}
	h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)
}

// Seeding lists torrents that are seeding
func (h *Handler) Seeding(ud tgbotapi.Update, cmd string) {
	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*seeding:* "+err.Error(), cmd)
		return
	}

	buf := new(bytes.Buffer)
	for i := range torrents {
		if torrents[i].Status == transmission.StatusSeeding ||
			torrents[i].Status == transmission.StatusSeedPending {
			buf.WriteString(h.FormatListLine(cmd, "<%d> %s\n", torrents[i].ID, torrents[i].Name))
		}
	}

	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "No torrents seeding", cmd)
		return
	}

	h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)
}

// Paused lists torrents that are paused
func (h *Handler) Paused(ud tgbotapi.Update, cmd string) {
	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*paused:* "+err.Error(), cmd)
		return
	}

	buf := new(bytes.Buffer)
	for i := range torrents {
		if torrents[i].Status == transmission.StatusStopped {
			buf.WriteString(h.FormatListLine(cmd, "<%d> %s\n", torrents[i].ID, torrents[i].Name))
		}
	}

	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "No paused torrents", cmd)
		return
	}

	h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)
}

// Checking lists torrents that are checking/verifying
func (h *Handler) Checking(ud tgbotapi.Update, cmd string) {
	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*checking:* "+err.Error(), cmd)
		return
	}

	buf := new(bytes.Buffer)
	for i := range torrents {
		if torrents[i].Status == transmission.StatusChecking ||
			torrents[i].Status == transmission.StatusCheckPending {
			buf.WriteString(h.FormatListLine(cmd, "<%d> %s\n", torrents[i].ID, torrents[i].Name))
		}
	}

	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "No torrents verifying", cmd)
		return
	}

	h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)
}

// Active lists torrents that are actively downloading or uploading
func (h *Handler) Active(ud tgbotapi.Update, cmd string) {
	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*active:* "+err.Error(), cmd)
		return
	}

	buf := new(bytes.Buffer)
	for i := range torrents {
		if torrents[i].RateDownload > 0 ||
			torrents[i].RateUpload > 0 {
			// escape markdown
			torrentName := h.Replacer.Replace(torrents[i].Name)
			buf.WriteString(h.FormatListLine(cmd, "`<%d>` *%s*\n%s ↓ *%s*  ↑ *%s* R: *%s*\n\n",
				torrents[i].ID, torrentName, torrents[i].TorrentStatus(),
				humanize.Bytes(torrents[i].RateDownload), humanize.Bytes(torrents[i].RateUpload),
				torrents[i].Ratio()))
		}
	}
	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "No active torrents", cmd)
		return
	}

	msgID := h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)

	if h.NoLive {
		return
	}

	// keep the active list live for 'duration * interval'
	for i := 0; i < h.Duration; i++ {
		time.Sleep(time.Second * h.Interval)
		// reset the buffer to reuse it
		buf.Reset()

		// update torrents
		torrents, err = h.Client.GetTorrents()
		if err != nil {
			continue // if there was error getting torrents, skip to the next iteration
		}

		// do the same loop again
		for i := range torrents {
			if torrents[i].RateDownload > 0 ||
				torrents[i].RateUpload > 0 {
				torrentName := h.Replacer.Replace(torrents[i].Name)
				buf.WriteString(h.FormatListLine(cmd, "`<%d>` *%s*\n%s ↓ *%s*  ↑ *%s* R: *%s*\n\n",
					torrents[i].ID, torrentName, torrents[i].TorrentStatus(),
					humanize.Bytes(torrents[i].RateDownload), humanize.Bytes(torrents[i].RateUpload),
					torrents[i].Ratio()))
			}
		}

		// no need to check if it is empty, as if the buffer is empty telegram won't change the message
		editConf := tgbotapi.NewEditMessageText(ud.Message.Chat.ID, msgID, buf.String())
		editConf.ParseMode = tgbotapi.ModeMarkdown
		h.Bot.Send(editConf)
	}
	// sleep one more time before putting the dashes
	time.Sleep(time.Second * h.Interval)

	// replace the speed with dashes to indicate that we are done being live
	buf.Reset()
	for i := range torrents {
		if torrents[i].RateDownload > 0 ||
			torrents[i].RateUpload > 0 {
			// escape markdown (dashes line has different placeholder count than output_string, use fmt.Sprintf)
			torrentName := h.Replacer.Replace(torrents[i].Name)
			buf.WriteString(fmt.Sprintf("`<%d>` *%s*\n%s ↓ *-*  ↑ *-* R: *-*\n\n",
				torrents[i].ID, torrentName, torrents[i].TorrentStatus()))
		}
	}

	editConf := tgbotapi.NewEditMessageText(ud.Message.Chat.ID, msgID, buf.String())
	editConf.ParseMode = tgbotapi.ModeMarkdown
	h.Bot.Send(editConf)
}

// Errors lists torrents with errors
func (h *Handler) Errors(ud tgbotapi.Update, cmd string) {
	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*errors:* "+err.Error(), cmd)
		return
	}

	buf := new(bytes.Buffer)
	for i := range torrents {
		if torrents[i].Error != 0 {
			buf.WriteString(h.FormatListLine(cmd, "<%d> %s\n%s\n\n", torrents[i].ID, torrents[i].Name, torrents[i].ErrorString))
		}
	}
	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "No errors", cmd)
		return
	}
	h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)
}

// Sort changes how torrents are sorted
func (h *Handler) Sort(ud tgbotapi.Update, tokens []string, cmd string) {
	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, `*sort* takes one of:
(*id, name, age, size, progress, downspeed, upspeed, download, upload, ratio*)
optionally start with (*rev*) for reversed order
e.g. "*sort rev size*" to get biggest torrents first.`, cmd)
		return
	}

	var reversed bool
	if strings.ToLower(tokens[0]) == "rev" {
		reversed = true
		tokens = tokens[1:]
	}

	switch strings.ToLower(tokens[0]) {
	case "id":
		if reversed {
			h.Client.GetTorrents()
		} else {
			h.Client.GetTorrents()
		}
		h.SendWithFormat(ud.Message.Chat.ID, "*sort:* "+tokens[0], cmd)
	case "name":
		h.SendWithFormat(ud.Message.Chat.ID, "*sort:* "+tokens[0], cmd)
	case "age":
		h.SendWithFormat(ud.Message.Chat.ID, "*sort:* "+tokens[0], cmd)
	case "size":
		h.SendWithFormat(ud.Message.Chat.ID, "*sort:* "+tokens[0], cmd)
	case "progress":
		h.SendWithFormat(ud.Message.Chat.ID, "*sort:* "+tokens[0], cmd)
	case "downspeed":
		h.SendWithFormat(ud.Message.Chat.ID, "*sort:* "+tokens[0], cmd)
	case "upspeed":
		h.SendWithFormat(ud.Message.Chat.ID, "*sort:* "+tokens[0], cmd)
	case "download":
		h.SendWithFormat(ud.Message.Chat.ID, "*sort:* "+tokens[0], cmd)
	case "upload":
		h.SendWithFormat(ud.Message.Chat.ID, "*sort:* "+tokens[0], cmd)
	case "ratio":
		h.SendWithFormat(ud.Message.Chat.ID, "*sort:* "+tokens[0], cmd)
	default:
		h.SendWithFormat(ud.Message.Chat.ID, "unknown sorting method", cmd)
		return
	}

	if reversed {
		h.SendWithFormat(ud.Message.Chat.ID, "*sort:* reversed "+tokens[0], cmd)
	}
}

// Trackers lists all trackers and their torrent count
func (h *Handler) Trackers(ud tgbotapi.Update, cmd string) {
	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*trackers:* "+err.Error(), cmd)
		return
	}

	trackers := make(map[string]int)

	for i := range torrents {
		for _, tracker := range torrents[i].Trackers {
			if matches := trackerRegex.FindStringSubmatch(tracker.Announce); matches != nil {
				trackers[matches[1]]++
			}
		}
	}

	buf := new(bytes.Buffer)
	for k, v := range trackers {
		buf.WriteString(h.FormatListLine(cmd, "%d - %s\n", v, k))
	}

	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "No trackers!", cmd)
		return
	}
	h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)
}

// trackerRegex extracts tracker hostname from URL
var trackerRegex = regexp.MustCompile(`[https?|udp]://([^:/]*)`)

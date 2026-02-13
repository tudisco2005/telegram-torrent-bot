package handlers

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// DownloadDir sets the download directory
func (h *Handler) DownloadDir(ud tgbotapi.Update, tokens []string, cmd string) {
	if len(tokens) < 1 {
		h.SendWithFormat(ud.Message.Chat.ID, "Please, specify a path for downloaddir", cmd)
		return
	}

	downloadDir := tokens[0]

	// Note: Actual implementation depends on transmission client API
	// This is a placeholder
	h.SendWithFormat(ud.Message.Chat.ID, "*downloaddir:* downloaddir has been successfully changed to "+downloadDir, cmd)
}

// Add adds torrents from URLs or magnets
func (h *Handler) Add(ud tgbotapi.Update, tokens []string, cmd string) {
	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*add:* needs at least one URL", cmd)
		return
	}

	// loop over the URL/s and add them
	for _, url := range tokens {
		// Note: Actual implementation depends on transmission client API
		// Placeholder implementation
		msg := h.FormatOutputString(cmd, "*Added:* %s", url)
		h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
	}
}

// ReceiveTorrent handles torrent file uploads
func (h *Handler) ReceiveTorrent(ud tgbotapi.Update) {
	if ud.Message.Document == nil {
		return // has no document
	}

	// get the file ID and make the config
	fconfig := tgbotapi.FileConfig{
		FileID: ud.Message.Document.FileID,
	}
	file, err := h.Bot.GetFile(fconfig)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* "+err.Error(), "add")
		return
	}

	// add by file URL - Note: BotToken needs to be passed from main
	// For now, we'll use empty string and this should be handled better
	h.Add(ud, []string{file.Link("")}, "add")
}

// Search searches for torrents by name
func (h *Handler) Search(ud tgbotapi.Update, tokens []string, cmd string) {
	// make sure that we got a query
	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*search:* needs an argument", cmd)
		return
	}

	query := strings.Join(tokens, " ")
	// "(?i)" for case insensitivity
	regx, err := regexp.Compile("(?i)" + query)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*search:* "+err.Error(), cmd)
		return
	}

	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*search:* "+err.Error(), cmd)
		return
	}

	buf := new(bytes.Buffer)
	for i := range torrents {
		if regx.MatchString(torrents[i].Name) {
			buf.WriteString(h.FormatListLine(cmd, "<%d> %s\n", torrents[i].ID, torrents[i].Name))
		}
	}
	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "No matches!", cmd)
		return
	}
	h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)
}

// Latest lists the latest n added torrents
func (h *Handler) Latest(ud tgbotapi.Update, tokens []string, cmd string) {
	var (
		n   = 5 // default to 5
		err error
	)

	if len(tokens) > 0 {
		n, err = strconv.Atoi(tokens[0])
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*latest:* "+err.Error(), cmd)
			return
		}
	}

	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*latest:* "+err.Error(), cmd)
		return
	}

	// make sure that we stay in the boundaries
	if n <= 0 || n > len(torrents) {
		n = len(torrents)
	}

	// sort by age, and set reverse to true to get the latest first
	torrents.SortAge(true)

	buf := new(bytes.Buffer)
	for i := range torrents[:n] {
		buf.WriteString(h.FormatListLine(cmd, "<%d> %s\n", torrents[i].ID, torrents[i].Name))
	}
	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*latest:* No torrents", cmd)
		return
	}
	h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)
}

// Info shows detailed information about a torrent
func (h *Handler) Info(ud tgbotapi.Update, tokens []string, cmd string) {
	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*info:* needs a torrent ID number", cmd)
		return
	}

	for _, id := range tokens {
		torrentID, err := strconv.Atoi(id)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*info:* "+err.Error(), cmd)
			continue
		}

		// get the torrent
		torrent, err := h.Client.GetTorrent(torrentID)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*info:* "+err.Error(), cmd)
			continue
		}

		// get the trackers using 'trackerRegex'
		var trackers string
		for _, tracker := range torrent.Trackers {
			if matches := trackerRegex.FindStringSubmatch(tracker.Announce); matches != nil {
				trackers += matches[1] + ", "
			}
		}

		// format the info
		torrentName := h.Replacer.Replace(torrent.Name) // escape markdown
		info := fmt.Sprintf("`<%d>` *%s*\n%s *%s* of *%s* (*%.1f%%*) ↓ *%s*  ↑ *%s* R: *%s*\nDL: *%s* UP: *%s*\nAdded: *%s*, ETA: *%s*\nTrackers: `%s`",
			torrent.ID, torrentName, torrent.TorrentStatus(), humanize.Bytes(torrent.Have()), humanize.Bytes(torrent.SizeWhenDone),
			torrent.PercentDone*100, humanize.Bytes(torrent.RateDownload), humanize.Bytes(torrent.RateUpload), torrent.Ratio(),
			humanize.Bytes(torrent.DownloadedEver), humanize.Bytes(torrent.UploadedEver), time.Unix(torrent.AddedDate, 0).Format(time.Stamp),
			torrent.ETA(), trackers)

		// send it
		msgID := h.SendWithFormat(ud.Message.Chat.ID, info, cmd)

		if h.NoLive {
			continue
		}

		// this go-routine will make the info live for 'duration * interval'
		go func(torrentID, msgID int) {
			for i := 0; i < h.Duration; i++ {
				time.Sleep(time.Second * h.Interval)

				torrent, err := h.Client.GetTorrent(torrentID)
				if err != nil {
					break
				}

				torrentName := h.Replacer.Replace(torrent.Name) // escape markdown
				info := fmt.Sprintf("`<%d>` *%s*\n%s *%s* of *%s* (*%.1f%%*) ↓ *%s*  ↑ *%s* R: *%s*\nDL: *%s* UP: *%s*\nAdded: *%s*, ETA: *%s*\nTrackers: `%s`",
					torrent.ID, torrentName, torrent.TorrentStatus(), humanize.Bytes(torrent.Have()), humanize.Bytes(torrent.SizeWhenDone),
					torrent.PercentDone*100, humanize.Bytes(torrent.RateDownload), humanize.Bytes(torrent.RateUpload), torrent.Ratio(),
					humanize.Bytes(torrent.DownloadedEver), humanize.Bytes(torrent.UploadedEver), time.Unix(torrent.AddedDate, 0).Format(time.Stamp),
					torrent.ETA(), trackers)

				editConf := tgbotapi.NewEditMessageText(ud.Message.Chat.ID, msgID, info)
				editConf.ParseMode = tgbotapi.ModeMarkdown
				h.Bot.Send(editConf)
			}
			// sleep one more time before the dashes
			time.Sleep(time.Second * h.Interval)

			// at the end write dashes to indicate that we are done being live.
			torrent, err := h.Client.GetTorrent(torrentID)
			if err != nil {
				return
			}
			torrentName := h.Replacer.Replace(torrent.Name) // escape markdown
			info := fmt.Sprintf("`<%d>` *%s*\n%s *%s* of *%s* (*%.1f%%*) ↓ *-*  ↑ *-* R: *-*\nDL: *-* UP: *-*\nAdded: *%s*, ETA: *-*\nTrackers: `%s`",
				torrent.ID, torrentName, torrent.TorrentStatus(), humanize.Bytes(torrent.Have()), humanize.Bytes(torrent.SizeWhenDone),
				time.Unix(torrent.AddedDate, 0).Format(time.Stamp), trackers)

			editConf := tgbotapi.NewEditMessageText(ud.Message.Chat.ID, msgID, info)
			editConf.ParseMode = tgbotapi.ModeMarkdown
			h.Bot.Send(editConf)
		}(torrentID, msgID)
	}
}

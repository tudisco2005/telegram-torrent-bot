package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/pyed/transmission"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// DownloadDir sets the download directory
func (h *Handler) DownloadDir(ud tgbotapi.Update, tokens []string, cmd string) {
	if len(tokens) < 1 {
		h.SendWithFormat(ud.Message.Chat.ID, "Please, specify a path for downloaddir", cmd)
		return
	}

	downloadDir := tokens[0]

	// Update in-memory download directory (session-wide). Persisting or applying
	// to Transmission per-torrent options is not implemented here.
	h.DefaultDownloadLocation = downloadDir
	msg := h.FormatOutputString(cmd, downloadDir)
	h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
}

// Add adds torrents from URLs or magnets via Transmission
func (h *Handler) Add(ud tgbotapi.Update, tokens []string, cmd string) {
	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*add:* needs at least one URL or magnet", cmd)
		return
	}

	var buf bytes.Buffer
	for _, url := range tokens {
		addCmd := transmission.NewAddCmdByURL(url)
		added, err := h.Client.ExecuteAddCommand(addCmd)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*add:* "+err.Error()+" — "+url, cmd)
			continue
		}

		// Debug: log the returned added struct and URL to help diagnose multiple empty names
		h.Logger.Printf("[DEBUG] Add: url=%s added=%#v", url, added)

		// collect successful adds and send a single combined message later
		buf.WriteString(h.FormatOutputString(cmd, added.Name))
		// ensure newline separation when FormatOutputString does not include it
		if !strings.HasSuffix(buf.String(), "\n") {
			buf.WriteString("\n")
		}
	}

	if buf.Len() > 0 {
		// Trim trailing newline
		out := strings.TrimRight(buf.String(), "\n")
		h.SendWithFormat(ud.Message.Chat.ID, out, cmd)
	}
}

// ReceiveTorrent handles torrent file uploads: saves to DEFAULT_TORRENT_LOCATION then adds to Transmission
func (h *Handler) ReceiveTorrent(ud tgbotapi.Update) {
	if ud.Message.Document == nil {
		return
	}
	h.Logger.Printf("[DEBUG] ReceiveTorrent: document received FileID=%s FileName=%q Size=%d",
		ud.Message.Document.FileID, ud.Message.Document.FileName, ud.Message.Document.FileSize)

	if h.DefaultTorrentLocation == "" {
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* DEFAULT_TORRENT_LOCATION is not set", "add")
		return
	}

	fconfig := tgbotapi.FileConfig{FileID: ud.Message.Document.FileID}
	file, err := h.Bot.GetFile(fconfig)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* "+err.Error(), "add")
		return
	}
	h.Logger.Printf("[DEBUG] ReceiveTorrent: GetFile ok FilePath=%s", file.FilePath)

	// Download file from Telegram
	downloadURL := file.Link(h.BotToken)
	h.Logger.Printf("[DEBUG] ReceiveTorrent: downloading from Telegram (FilePath=%s)", file.FilePath)
	resp, err := http.Get(downloadURL)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* download failed: "+err.Error(), "add")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* download failed: status "+resp.Status, "add")
		return
	}
	h.Logger.Printf("[DEBUG] ReceiveTorrent: download ok status=%d content_length=%d", resp.StatusCode, resp.ContentLength)

	// Ensure save directory exists
	h.Logger.Printf("[DEBUG] ReceiveTorrent: ensuring directory exists: %s", h.DefaultTorrentLocation)
	if err := os.MkdirAll(h.DefaultTorrentLocation, 0755); err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* "+err.Error(), "add")
		return
	}

	// Save to DEFAULT_TORRENT_LOCATION (use document filename or fallback)
	name := ud.Message.Document.FileName
	if name == "" {
		name = filepath.Base(file.FilePath)
	}
	if name == "" || name == "." {
		name = "torrent_" + ud.Message.Document.FileID + ".torrent"
	}
	if !strings.HasSuffix(strings.ToLower(name), ".torrent") {
		name = name + ".torrent"
	}
	savePath := filepath.Join(h.DefaultTorrentLocation, name)
	h.Logger.Printf("[DEBUG] ReceiveTorrent: saving .torrent as %s", savePath)

	out, err := os.Create(savePath)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* "+err.Error(), "add")
		return
	}
	n, err := io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(savePath)
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* "+err.Error(), "add")
		return
	}
	h.Logger.Printf("[DEBUG] ReceiveTorrent: saved %d bytes to %s", n, savePath)

	// Add to Transmission from the saved file
	h.Logger.Printf("[DEBUG] ReceiveTorrent: adding to Transmission from file %s", savePath)
	addCmd, err := transmission.NewAddCmdByFile(savePath)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* "+err.Error(), "add")
		return
	}
	added, err := h.Client.ExecuteAddCommand(addCmd)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* "+err.Error(), "add")
		return
	}
	h.Logger.Printf("[DEBUG] ReceiveTorrent: added to Transmission id=%d name=%q", added.ID, added.Name)
	msg := h.FormatOutputString("add", added.Name)
	h.SendWithFormat(ud.Message.Chat.ID, msg, "add")
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
			buf.WriteString(h.FormatOutputString(cmd, torrents[i].ID, torrents[i].Name))
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
		buf.WriteString(h.FormatOutputString(cmd, torrents[i].ID, torrents[i].Name))
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

		if h.NoLive || h.UpdateMaxIterations == 0 {
			continue
		}

		iterations := h.Duration
		if h.UpdateMaxIterations > 0 && h.UpdateMaxIterations < iterations {
			iterations = h.UpdateMaxIterations
		}
		chatID := ud.Message.Chat.ID
		// this go-routine will make the info live for 'iterations * interval'
		go func(torrentID, msgID int, chatID int64) {
			for i := 0; i < iterations; i++ {
				time.Sleep(h.Interval)

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

				editConf := tgbotapi.NewEditMessageText(chatID, msgID, info)
				editConf.ParseMode = tgbotapi.ModeMarkdown
				if resp, err := h.Bot.Send(editConf); err != nil {
					h.Logger.Printf("[DEBUG] EditMessage failed: ChatID=%d MsgID=%d Len=%d Err=%v", chatID, msgID, len(info), err)
				} else {
					h.Logger.Printf("[DEBUG] EditMessage sent: ChatID=%d MsgID=%d RespMessageID=%d", chatID, msgID, resp.MessageID)
				}
			}
			// sleep one more time before the dashes
			time.Sleep(h.Interval)

			// at the end write dashes to indicate that we are done being live.
			torrent, err := h.Client.GetTorrent(torrentID)
			if err != nil {
				return
			}
			torrentName := h.Replacer.Replace(torrent.Name) // escape markdown
			info := fmt.Sprintf("`<%d>` *%s*\n%s *%s* of *%s* (*-%%*) ↓ *-*  ↑ *-* R: *-*\nDL: *-* UP: *-*\nAdded: *%s*, ETA: *-*\nTrackers: `%s`",
				torrent.ID, torrentName, torrent.TorrentStatus(), humanize.Bytes(torrent.Have()), humanize.Bytes(torrent.SizeWhenDone),
				time.Unix(torrent.AddedDate, 0).Format(time.Stamp), trackers)

			editConf := tgbotapi.NewEditMessageText(chatID, msgID, info)
			editConf.ParseMode = tgbotapi.ModeMarkdown
			if resp, err := h.Bot.Send(editConf); err != nil {
				h.Logger.Printf("[DEBUG] EditMessage failed: ChatID=%d MsgID=%d Len=%d Err=%v", chatID, msgID, len(info), err)
			} else {
				h.Logger.Printf("[DEBUG] EditMessage sent: ChatID=%d MsgID=%d RespMessageID=%d", chatID, msgID, resp.MessageID)
			}
		}(torrentID, msgID, chatID)
	}
}

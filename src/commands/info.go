package commands

import (
	"fmt"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	"github.com/tudisco2005/telegram-torrent-bot/utils"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Info shows detailed information about a torrent
func Info(h *handlers.Handler, ud tgbotapi.Update, tokens []string, cmd string) {
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

		torrent, err := h.Client.GetTorrent(torrentID)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*info:* "+err.Error(), cmd)
			continue
		}

		// get the trackers using 'trackerRegex'
		var trackers string
		for _, tracker := range torrent.Trackers {
			if matches := utils.TrackerRegex.FindStringSubmatch(tracker.Announce); matches != nil {
				trackers += matches[1] + ", "
			}
		}

		torrentName := h.Replacer.Replace(torrent.Name)
		info := fmt.Sprintf("`<%d>` *%s*\n%s *%s* of *%s* (*%.1f%%*) ↓ *%s*  ↑ *%s* R: *%s*\nDL: *%s* UP: *%s*\nAdded: *%s*, ETA: *%s*\nTrackers: `%s`",
			torrent.ID, torrentName, torrent.TorrentStatus(), humanize.Bytes(torrent.Have()), humanize.Bytes(torrent.SizeWhenDone),
			torrent.PercentDone*100, humanize.Bytes(torrent.RateDownload), humanize.Bytes(torrent.RateUpload), torrent.Ratio(),
			humanize.Bytes(torrent.DownloadedEver), humanize.Bytes(torrent.UploadedEver), time.Unix(torrent.AddedDate, 0).Format(time.Stamp),
			torrent.ETA(), trackers)

		msgID := h.SendWithFormat(ud.Message.Chat.ID, info, cmd)

		if h.NoLive || h.UpdateMaxIterations == 0 {
			continue
		}

		iterations := h.Duration
		if h.UpdateMaxIterations > 0 && h.UpdateMaxIterations < iterations {
			iterations = h.UpdateMaxIterations
		}
		chatID := ud.Message.Chat.ID
		ctx, finish := h.StartLiveTask(fmt.Sprintf("info:%d:%d", chatID, torrentID))

		go func(torrentID, msgID int, chatID int64) {
			defer finish()
			ticker := time.NewTicker(h.Interval)
			defer ticker.Stop()

			for i := 0; i < iterations; i++ {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}

				torrent, err := h.Client.GetTorrent(torrentID)
				if err != nil {
					break
				}

				torrentName := h.Replacer.Replace(torrent.Name)
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

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			torrent, err := h.Client.GetTorrent(torrentID)
			if err != nil {
				return
			}
			torrentName := h.Replacer.Replace(torrent.Name)
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

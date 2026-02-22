package commands

import (
	"bytes"
	"fmt"
	"time"

	tgbotapi "gopkg.in/telegram-bot-api.v4"

	"github.com/dustin/go-humanize"
	"github.com/pyed/transmission"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
)

// Active lists torrents that are actively downloading or uploading
func Active(h *handlers.Handler, ud tgbotapi.Update, cmd string) {
	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*active:* "+err.Error(), cmd)
		return
	}

	buf := new(bytes.Buffer)
	for i := range torrents {
		if torrents[i].RateDownload > 0 ||
			torrents[i].RateUpload > 0 {

			torrentName := h.Replacer.Replace(torrents[i].Name)
			buf.WriteString(h.FormatOutputString(cmd,
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

	if h.NoLive || h.UpdateMaxIterations == 0 {
		return
	}

	iterations := h.Duration
	if h.UpdateMaxIterations > 0 && h.UpdateMaxIterations < iterations {
		iterations = h.UpdateMaxIterations
	}
	chatID := ud.Message.Chat.ID
	ctx, finish := h.StartLiveTask(fmt.Sprintf("active:%d", chatID))

	go func() {
		defer finish()
		ticker := time.NewTicker(h.Interval)
		defer ticker.Stop()

		var torrents transmission.Torrents
		liveBuf := new(bytes.Buffer)
		for i := 0; i < iterations; i++ {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			liveBuf.Reset()

			var err error
			torrents, err = h.Client.GetTorrents()
			if err != nil {
				continue
			}

			for j := range torrents {
				if torrents[j].RateDownload > 0 || torrents[j].RateUpload > 0 {
					torrentName := h.Replacer.Replace(torrents[j].Name)
					liveBuf.WriteString(h.FormatOutputString(cmd,
						torrents[j].ID, torrentName, torrents[j].TorrentStatus(),
						humanize.Bytes(torrents[j].RateDownload), humanize.Bytes(torrents[j].RateUpload),
						torrents[j].Ratio()))
				}
			}

			editConf := tgbotapi.NewEditMessageText(chatID, msgID, liveBuf.String())
			editConf.ParseMode = tgbotapi.ModeMarkdown
			if resp, err := h.Bot.Send(editConf); err != nil {
				h.Logger.Printf("[DEBUG] EditMessage failed: ChatID=%d MsgID=%d Len=%d Err=%v", chatID, msgID, len(liveBuf.String()), err)
			} else {
				h.Logger.Printf("[DEBUG] EditMessage sent: ChatID=%d MsgID=%d RespMessageID=%d", chatID, msgID, resp.MessageID)
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		liveBuf.Reset()
		torrents, _ = h.Client.GetTorrents()
		for i := range torrents {
			if torrents[i].RateDownload > 0 || torrents[i].RateUpload > 0 {
				torrentName := h.Replacer.Replace(torrents[i].Name)
				liveBuf.WriteString(fmt.Sprintf("`<%d>` *%s*\n%s ↓ *-*  ↑ *-* R: *-*\n\n",
					torrents[i].ID, torrentName, torrents[i].TorrentStatus()))
			}
		}
		editConf := tgbotapi.NewEditMessageText(chatID, msgID, liveBuf.String())
		editConf.ParseMode = tgbotapi.ModeMarkdown
		if resp, err := h.Bot.Send(editConf); err != nil {
			h.Logger.Printf("[DEBUG] EditMessage failed: ChatID=%d MsgID=%d Len=%d Err=%v", chatID, msgID, len(liveBuf.String()), err)
		} else {
			h.Logger.Printf("[DEBUG] EditMessage sent: ChatID=%d MsgID=%d RespMessageID=%d", chatID, msgID, resp.MessageID)
		}
	}()
}

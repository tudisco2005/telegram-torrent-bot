package commands

import (
	"bytes"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Head lists the first n torrents (default: 5)
func Head(h *handlers.Handler, ud tgbotapi.Update, tokens []string, cmd string) {
	var (
		n   = 5 // default to 5
		err error
	)

	if len(tokens) > 0 {
		n, err = strconv.Atoi(tokens[0])
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*head:* "+err.Error(), cmd)
			return
		}
	}

	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*head:* "+err.Error(), cmd)
		return
	}

	if n <= 0 || n > len(torrents) {
		n = len(torrents)
	}

	buf := new(bytes.Buffer)
	for i := range torrents[:n] {
		torrentName := h.Replacer.Replace(torrents[i].Name)
		buf.WriteString(h.FormatOutputString(cmd,
			torrents[i].ID, torrentName, torrents[i].TorrentStatus(), humanize.Bytes(torrents[i].Have()),
			humanize.Bytes(torrents[i].SizeWhenDone), torrents[i].PercentDone*100, humanize.Bytes(torrents[i].RateDownload),
			humanize.Bytes(torrents[i].RateUpload), torrents[i].Ratio()))
	}

	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*head:* no torrents", cmd)
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

	go func() {
		liveBuf := new(bytes.Buffer)
		for i := 0; i < iterations; i++ {
			time.Sleep(h.Interval)
			liveBuf.Reset()

			torrents, err := h.Client.GetTorrents()
			if err != nil || len(torrents) < 1 {
				continue
			}
			nn := n
			if nn <= 0 || nn > len(torrents) {
				nn = len(torrents)
			}
			for _, torrent := range torrents[:nn] {
				torrentName := h.Replacer.Replace(torrent.Name)
				liveBuf.WriteString(h.FormatOutputString(cmd,
					torrent.ID, torrentName, torrent.TorrentStatus(), humanize.Bytes(torrent.Have()),
					humanize.Bytes(torrent.SizeWhenDone), torrent.PercentDone*100, humanize.Bytes(torrent.RateDownload),
					humanize.Bytes(torrent.RateUpload), torrent.Ratio()))
			}
			editConf := tgbotapi.NewEditMessageText(chatID, msgID, liveBuf.String())
			editConf.ParseMode = tgbotapi.ModeMarkdown
			if resp, err := h.Bot.Send(editConf); err != nil {
				h.Logger.Printf("[DEBUG] EditMessage failed: ChatID=%d MsgID=%d Len=%d Err=%v", chatID, msgID, len(liveBuf.String()), err)
			} else {
				h.Logger.Printf("[DEBUG] EditMessage sent: ChatID=%d MsgID=%d RespMessageID=%d", chatID, msgID, resp.MessageID)
			}
		}
	}()
}

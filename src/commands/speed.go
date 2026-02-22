package commands

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Speed shows current download and upload speeds
func Speed(h *handlers.Handler, ud tgbotapi.Update, cmd string) {
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
	ctx, finish := h.StartLiveTask(fmt.Sprintf("speed:%d", chatID))

	go func() {
		defer finish()
		ticker := time.NewTicker(h.Interval)
		defer ticker.Stop()

		for i := 0; i < iterations; i++ {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
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
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		editConf := tgbotapi.NewEditMessageText(chatID, msgID, "↓ - B  ↑ - B")
		if resp, err := h.Bot.Send(editConf); err != nil {
			h.Logger.Printf("[DEBUG] EditMessage failed: ChatID=%d MsgID=%d Len=%d Err=%v", chatID, msgID, len("↓ - B  ↑ - B"), err)
		} else {
			h.Logger.Printf("[DEBUG] EditMessage sent: ChatID=%d MsgID=%d RespMessageID=%d", chatID, msgID, resp.MessageID)
		}
	}()
}

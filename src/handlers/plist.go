package handlers

import (
	"bytes"
	"strings"
	"time"

	"github.com/pyed/transmission"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Plist shows a pretty list with progress bars and status emojis
func (h *Handler) Plist(ud tgbotapi.Update, tokens []string, cmd string) {
	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*plist:* "+err.Error(), cmd)
		return
	}

	if len(torrents) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "No torrents", cmd)
		return
	}

	buf := new(bytes.Buffer)

	// optional filter: if tokens given, filter by name (case-insensitive substring)
	var filter string
	if len(tokens) > 0 {
		filter = strings.ToLower(strings.Join(tokens, " "))
	}

	const barLen = 15

	for i := range torrents {
		t := torrents[i]
		if filter != "" && !strings.Contains(strings.ToLower(t.Name), filter) {
			continue
		}

		// determine emoji based on state
		var statusEmoji string
		if t.Error != 0 {
			statusEmoji = "❌"
		} else if t.PercentDone >= 1.0 {
			statusEmoji = "✅"
		} else if t.Status == transmission.StatusDownloading || t.Status == transmission.StatusDownloadPending {
			statusEmoji = "⬇️"
		} else if t.Status == transmission.StatusStopped {
			statusEmoji = "⏸️"
		} else {
			statusEmoji = "❓"
		}

		// build progress bar
		filled := int(t.PercentDone*float64(barLen) + 0.5)
		if filled < 0 {
			filled = 0
		}
		if filled > barLen {
			filled = barLen
		}
		bar := "[" + strings.Repeat("=", filled) + strings.Repeat(" ", barLen-filled) + "]"

		// percent as integer
		pct := int(t.PercentDone * 100)

		// ETA (use torrent.ETA() if available, otherwise -)
		eta := t.ETA()
		if eta == "" {
			eta = "-"
		}

		// escape markdown name
		name := h.Replacer.Replace(t.Name)

		// Use the output_string template from commands.json via FormatOutputString
		buf.WriteString(h.FormatOutputString(cmd, t.ID, name, statusEmoji, bar, pct, eta))
	}

	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "No matches", cmd)
		return
	}

	msgID := h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)

	// live updates: refresh progress bar for a while
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

			ts, err := h.Client.GetTorrents()
			if err != nil {
				continue
			}

			for j := range ts {
				t := ts[j]
				if filter != "" && !strings.Contains(strings.ToLower(t.Name), filter) {
					continue
				}

				var statusEmoji string
				if t.Error != 0 {
					statusEmoji = "❌"
				} else if t.PercentDone >= 1.0 {
					statusEmoji = "✅"
				} else if t.Status == transmission.StatusDownloading || t.Status == transmission.StatusDownloadPending {
					statusEmoji = "⬇️"
				} else if t.Status == transmission.StatusStopped {
					statusEmoji = "⏸️"
				} else {
					statusEmoji = "❓"
				}

				filled := int(t.PercentDone*float64(barLen) + 0.5)
				if filled < 0 {
					filled = 0
				}
				if filled > barLen {
					filled = barLen
				}
				bar := "[" + strings.Repeat("=", filled) + strings.Repeat(" ", barLen-filled) + "]"
				pct := int(t.PercentDone * 100)
				eta := t.ETA()
				if eta == "" {
					eta = "-"
				}
				name := h.Replacer.Replace(t.Name)
				liveBuf.WriteString(h.FormatOutputString(cmd, t.ID, name, statusEmoji, bar, pct, eta))
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

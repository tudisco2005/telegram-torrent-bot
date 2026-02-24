package commands

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pyed/transmission"
	"github.com/tudisco2005/telegram-torrent-bot/commands/helpers"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	"github.com/tudisco2005/telegram-torrent-bot/utils"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Plist shows a pretty list with progress bars and status emojis
func Plist(h *handlers.Handler, ud tgbotapi.Update, tokens []string, cmd string) {
	mode := "downloading"
	if len(tokens) > 0 {
		switch strings.ToLower(tokens[0]) {
		case "?", "help":
			help := "plist usage:\n" +
				"- `pls` : list downloading torrents\n" +
				"- `pls all` : list all torrents (downloading, paused, complete)\n" +
				"- `pls stopped` : list paused/stopped downloads\n" +
				"- `pls ?` : show this help"
			h.SendWithFormat(ud.Message.Chat.ID, help, cmd)
			return
		case "all":
			mode = "all"
			tokens = tokens[1:]
		case "stopped", "paused":
			mode = "stopped"
			tokens = tokens[1:]
		}
	}

	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*plist:* "+err.Error(), cmd)
		return
	}

	sorter, tokens, err := helpers.ParseInlineSort(tokens)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*plist:* "+err.Error(), cmd)
		return
	}
	if sorter != nil {
		sorter(torrents)
	} else if mode == "all" {
		sortAllByState(torrents)
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
	listedCount := 0

	for i := range torrents {
		t := torrents[i]
		if !plistMatchMode(t, mode) {
			continue
		}
		if filter != "" && !strings.Contains(strings.ToLower(t.Name), filter) {
			continue
		}
		listedCount++

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

		bar := utils.ProgressBar(t.PercentDone, barLen)

		pct := int(t.PercentDone * 100)

		eta := t.ETA()
		if eta == "" {
			eta = "-"
		}

		name := h.Replacer.Replace(t.Name)

		buf.WriteString(h.FormatOutputString(cmd, t.ID, name, statusEmoji, bar, pct, eta))
	}

	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "No matches", cmd)
		return
	}

	msgID, paged := h.SendWithPaginationFormat(ud.Message.Chat.ID, fmt.Sprintf("Listing %d torrents:\n%s", listedCount, buf.String()), cmd)
	if paged {
		return
	}

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
			liveListedCount := 0

			ts, err := h.Client.GetTorrents()
			if err != nil {
				continue
			}
			if sorter != nil {
				sorter(ts)
			} else if mode == "all" {
				sortAllByState(ts)
			}

			for j := range ts {
				t := ts[j]
				if !plistMatchMode(t, mode) {
					continue
				}
				if filter != "" && !strings.Contains(strings.ToLower(t.Name), filter) {
					continue
				}
				liveListedCount++

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

				bar := utils.ProgressBar(t.PercentDone, barLen)
				pct := int(t.PercentDone * 100)
				eta := t.ETA()
				if eta == "" {
					eta = "-"
				}
				name := h.Replacer.Replace(t.Name)
				liveBuf.WriteString(h.FormatOutputString(cmd, t.ID, name, statusEmoji, bar, pct, eta))
			}

			editConf := tgbotapi.NewEditMessageText(chatID, msgID, fmt.Sprintf("Listing %d torrents:\n%s", liveListedCount, liveBuf.String()))
			editConf.ParseMode = tgbotapi.ModeMarkdown
			if resp, err := h.Bot.Send(editConf); err != nil {
				h.Logger.Printf("[DEBUG] EditMessage failed: ChatID=%d MsgID=%d Len=%d Err=%v", chatID, msgID, len(liveBuf.String()), err)
			} else {
				h.Logger.Printf("[DEBUG] EditMessage sent: ChatID=%d MsgID=%d RespMessageID=%d", chatID, msgID, resp.MessageID)
			}
		}
	}()
}

func plistMatchMode(t *transmission.Torrent, mode string) bool {
	switch mode {
	case "all":
		return true
	case "stopped":
		return t.Status == transmission.StatusStopped && t.PercentDone < 1.0
	default: // downloading
		return t.PercentDone < 1.0 && t.Status != transmission.StatusStopped
	}
}

func sortAllByState(ts transmission.Torrents) {
	sort.SliceStable(ts, func(i, j int) bool {
		gi := plistStateGroup(ts[i])
		gj := plistStateGroup(ts[j])
		if gi != gj {
			return gi < gj
		}
		return ts[i].ID < ts[j].ID
	})
}

func plistStateGroup(t *transmission.Torrent) int {
	if t.PercentDone >= 1.0 {
		return 2
	}
	if t.Status == transmission.StatusStopped {
		return 1
	}
	return 0
}

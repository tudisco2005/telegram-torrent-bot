package commands

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// ListHandler lists torrents, optionally filtered by tracker
func List(h *handlers.Handler, ud tgbotapi.Update, tokens []string, cmd string) {
	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*list:* "+err.Error(), cmd)
		return
	}

	sorter, tokens, err := parseInlineSort(tokens)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*list:* "+err.Error(), cmd)
		return
	}
	if sorter != nil {
		sorter(torrents)
	}

	buf := new(bytes.Buffer)

	if len(tokens) != 0 {
		regx, err := regexp.Compile("(?i)" + tokens[0])
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*list:* "+err.Error(), cmd)
			return
		}

		for i := range torrents {
			for _, tracker := range torrents[i].Trackers {
				if regx.MatchString(tracker.Announce) {
					torrentName := h.Replacer.Replace(torrents[i].Name)
					buf.WriteString(h.FormatOutputString(cmd, torrents[i].ID, torrentName))
					break
				}
			}
		}
	} else {
		for i := range torrents {
			torrentName := h.Replacer.Replace(torrents[i].Name)
			buf.WriteString(h.FormatOutputString(cmd, torrents[i].ID, torrentName))
		}
	}

	if buf.Len() == 0 {
		if len(tokens) != 0 {
			h.SendWithFormat(ud.Message.Chat.ID, "*list:* no matches", cmd, "markdown")
			return
		}
		h.SendWithFormat(ud.Message.Chat.ID, "*list:* no torrents", cmd, "markdown")
		return
	}

	h.SendWithPaginationFormat(ud.Message.Chat.ID, fmt.Sprintf("Listing %d torrents:\n%s", len(torrents), buf.String()), cmd)
}

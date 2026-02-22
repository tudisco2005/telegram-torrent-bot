package commands

import (
	"bytes"
	"strings"

	"github.com/pyed/transmission"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	"github.com/tudisco2005/telegram-torrent-bot/utils"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Add adds torrents from URLs or magnets via Transmission
func Add(h *handlers.Handler, ud tgbotapi.Update, tokens []string, cmd string) {
	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*add:* needs at least one URL or magnet", cmd)
		return
	}

	var buf bytes.Buffer

	existingTorrents, _ := h.Client.GetTorrents()
	existingIDs := make(map[int]bool)
	for i := range existingTorrents {
		existingIDs[existingTorrents[i].ID] = true
	}

	for _, link := range tokens {

		if !strings.HasPrefix(link, "magnet") && !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
			buf.WriteString("*add:* invalid URL or magnet link — " + link + "\n")
			continue
		}

		addCmd := transmission.NewAddCmdByURL(link)
		h.Logger.Printf("[DEBUG] Add: attempting to add link=%s", link)
		added, err := h.Client.ExecuteAddCommand(addCmd)
		if err != nil {
			buf.WriteString("*add:* " + err.Error() + " — " + link + "\n")
			continue
		}

		h.Logger.Printf("[DEBUG] Add: url=%s added=%#v existingID=%v", link, added, existingIDs[added.ID])

		metadataPending, blocked := validateAddedTorrent(
			h,
			ud.Message.Chat.ID,
			cmd,
			added.ID,
			link,
			"*Added magnet*: metadata not available yet — letting Transmission fetch metadata",
		)
		if metadataPending || blocked {
			continue
		}

		safeName := utils.EscapeFileMD(added.Name)
		buf.WriteString(h.FormatOutputString(cmd, safeName))

		if !strings.HasSuffix(buf.String(), "\n") {
			buf.WriteString("\n")
		}
	}

	if buf.Len() > 0 {

		out := strings.TrimRight(buf.String(), "\n")
		h.SendWithFormat(ud.Message.Chat.ID, out, cmd)
	}
}

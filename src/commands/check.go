package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	"github.com/tudisco2005/telegram-torrent-bot/utils"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Check verifies one or more torrents
func Check(h *handlers.Handler, ud tgbotapi.Update, tokens []string, cmd string) {

	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*check:* needs an argument", cmd)
		return
	}

	if tokens[0] == "all" {
		h.SendWithFormat(ud.Message.Chat.ID, "Verifying all torrents", cmd)

		torrents, err := h.Client.GetTorrents()
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*check:* "+err.Error(), cmd)
			return
		}

		okCount := 0
		errCount := 0
		errorLines := make([]string, 0)

		for _, t := range torrents {
			_, verifyErr := h.Client.VerifyTorrent(t.ID)
			if verifyErr != nil {
				errCount++
				errorLines = append(errorLines, fmt.Sprintf("`<%d>` %s `%s`", t.ID, utils.EscapeFileMD(t.Name), utils.EscapeMarkdown(verifyErr.Error())))
				continue
			}
			okCount++
		}

		h.SendWithFormat(ud.Message.Chat.ID, fmt.Sprintf("Checked torrents: %d ok, %d error", okCount, errCount), cmd)
		if errCount > 0 {
			h.SendWithFormat(ud.Message.Chat.ID, "List of torrents with errors (id:error):\n"+strings.Join(errorLines, "\n"), cmd)
		}
		return
	}

	for _, id := range tokens {
		num, err := strconv.Atoi(id)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*check:* "+err.Error(), cmd)
			continue
		}
		status, err := h.Client.VerifyTorrent(num)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*check:* "+err.Error(), cmd)
			continue
		}

		torrent, err := h.Client.GetTorrent(num)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*check:* "+err.Error(), cmd)
			continue
		}
		msg := h.FormatOutputString(cmd, status, torrent.Name)
		h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
	}
}

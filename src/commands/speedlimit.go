package commands

import (
	"fmt"
	"strconv"

	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// SpeedLimit sets either download or upload limit
func SpeedLimit(h *handlers.Handler, ud tgbotapi.Update, tokens []string, limitType string, cmd string) {
	if len(tokens) < 1 {
		h.SendWithFormat(ud.Message.Chat.ID, "Please, specify the limit", cmd)
		return
	}

	limit, err := strconv.ParseUint(tokens[0], 10, 32)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "Please, specify the limit as number of kilobytes", cmd)
		return
	}

	h.SendWithFormat(ud.Message.Chat.ID,
		fmt.Sprintf("*%s:* limit has been successfully changed to %d KB/s", limitType, limit), cmd)
}

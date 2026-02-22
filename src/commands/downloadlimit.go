package commands

import (
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// DownloadLimit sets the global download speed limit
func DownloadLimit(h *handlers.Handler, ud tgbotapi.Update, tokens []string, cmd string) {
	SpeedLimit(h, ud, tokens, "DownloadLimitType", cmd)
}

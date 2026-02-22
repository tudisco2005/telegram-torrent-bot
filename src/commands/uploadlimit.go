package commands

import (
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// UploadLimit sets the global upload speed limit
func UploadLimit(h *handlers.Handler, ud tgbotapi.Update, tokens []string, cmd string) {
	SpeedLimit(h, ud, tokens, "UploadLimitType", cmd)
}

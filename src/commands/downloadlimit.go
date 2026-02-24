package commands

import (
	"github.com/pyed/transmission"
	"github.com/tudisco2005/telegram-torrent-bot/commands/helpers"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// DownloadLimit sets the global download speed limit
func DownloadLimit(h *handlers.Handler, ud tgbotapi.Update, tokens []string, cmd string) {
	helpers.SpeedLimit(h, ud, tokens, transmission.DownloadLimitType, cmd)
}

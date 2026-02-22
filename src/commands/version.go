package commands

import (
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Version shows the version information (uses output_format and output_string for "version" from commands.json)
func Version(h *handlers.Handler, ud tgbotapi.Update, version string) {

	msg := h.FormatOutputString("version", h.Client.Version(), version)
	h.SendWithFormat(ud.Message.Chat.ID, msg, "version")
}

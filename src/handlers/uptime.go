package handlers

import (
	"strconv"
	"strings"
	"time"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Uptime shows the system uptime (reads /proc/uptime on Linux)
func (h *Handler) Uptime(ud tgbotapi.Update, cmd string) {
	// Prefer bot process start time if available
	var d time.Duration
	if !h.StartTime.IsZero() {
		d = time.Since(h.StartTime)
	} else {
		// error
		h.SendWithFormat(ud.Message.Chat.ID, "Uptime information not available", cmd)
		return
	}

	// Format as Xd Yh Zm Ws
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	secsInt := int(d.Seconds()) % 60

	var parts []string
	if days > 0 {
		parts = append(parts, strconv.Itoa(days)+"d")
	}
	if hours > 0 {
		parts = append(parts, strconv.Itoa(hours)+"h")
	}
	if mins > 0 {
		parts = append(parts, strconv.Itoa(mins)+"m")
	}
	parts = append(parts, strconv.Itoa(secsInt)+"s")

	formatted := strings.Join(parts, " ")

	msg := h.FormatOutputString(cmd, formatted)
	h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
}

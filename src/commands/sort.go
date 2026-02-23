package commands

import (
	"strings"

	"github.com/pyed/transmission"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

func sendSortHelp(h *handlers.Handler, chatID int64, cmd string) {
	h.SendWithFormat(chatID, `*sort* takes one of:
		(*id, name, age, size, progress, downspeed, upspeed, download, upload, ratio*)
		optionally start with (*rev*) for reversed order
		e.g. "*sort rev size*" to get biggest torrents first.`, cmd)
}

// Sort changes how torrents are sorted
func Sort(h *handlers.Handler, ud tgbotapi.Update, tokens []string, cmd string) {
	if len(tokens) == 0 {
		sendSortHelp(h, ud.Message.Chat.ID, cmd)
		return
	}
	if strings.ToLower(tokens[0]) == "?" || strings.ToLower(tokens[0]) == "help" {
		sendSortHelp(h, ud.Message.Chat.ID, cmd)
		return
	}

	var reversed bool
	if strings.ToLower(tokens[0]) == "rev" {
		reversed = true
		tokens = tokens[1:]
	}

	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "missing sorting method after rev", cmd)
		return
	}
	if strings.ToLower(tokens[0]) == "?" || strings.ToLower(tokens[0]) == "help" {
		sendSortHelp(h, ud.Message.Chat.ID, cmd)
		return
	}

	switch strings.ToLower(tokens[0]) {
	case "id":
		if reversed {
			h.Client.SetSort(transmission.SortRevID)
		} else {
			h.Client.SetSort(transmission.SortID)
		}
	case "name":
		if reversed {
			h.Client.SetSort(transmission.SortRevName)
		} else {
			h.Client.SetSort(transmission.SortName)
		}
	case "age":
		if reversed {
			h.Client.SetSort(transmission.SortRevAge)
		} else {
			h.Client.SetSort(transmission.SortAge)
		}
	case "size":
		if reversed {
			h.Client.SetSort(transmission.SortRevSize)
		} else {
			h.Client.SetSort(transmission.SortSize)
		}
	case "progress":
		if reversed {
			h.Client.SetSort(transmission.SortRevProgress)
		} else {
			h.Client.SetSort(transmission.SortProgress)
		}
	case "downspeed":
		if reversed {
			h.Client.SetSort(transmission.SortRevDownSpeed)
		} else {
			h.Client.SetSort(transmission.SortDownSpeed)
		}
	case "upspeed":
		if reversed {
			h.Client.SetSort(transmission.SortRevUpSpeed)
		} else {
			h.Client.SetSort(transmission.SortUpSpeed)
		}
	case "download":
		if reversed {
			h.Client.SetSort(transmission.SortRevDownloaded)
		} else {
			h.Client.SetSort(transmission.SortDownloaded)
		}
	case "upload":
		if reversed {
			h.Client.SetSort(transmission.SortRevUploaded)
		} else {
			h.Client.SetSort(transmission.SortUploaded)
		}
	case "ratio":
		if reversed {
			h.Client.SetSort(transmission.SortRevRatio)
		} else {
			h.Client.SetSort(transmission.SortRatio)
		}
	default:
		h.SendWithFormat(ud.Message.Chat.ID, "unknown sorting method", cmd)
		return
	}

	msg := "*sort:* " + tokens[0]
	if reversed {
		msg += " (reversed)"
	}
	h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
}

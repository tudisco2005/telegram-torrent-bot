package handlers

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/pyed/transmission"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Handler contains the necessary clients and configuration for handling commands
type Handler struct {
	Bot                   *tgbotapi.BotAPI
	Client                *transmission.TransmissionClient
	NoLive                bool
	Interval              time.Duration
	Duration              int
	Replacer              StringReplacer
	SendMessage           MessageSender
	Logger                Logger
	OutputFormatByCommand map[string]string // canonical command name -> "markdown" or "plain"
	OutputStringByCommand map[string]string // canonical command name -> format string for output
	ListOutputByCommand   map[string]bool   // when true, output_string is used to format each line of list output
}

// FormatListLine formats a single line of list output. When list_output is true for the command, uses output_string from JSON; otherwise uses the provided template.
func (h *Handler) FormatListLine(command string, defaultTemplate string, args ...interface{}) string {
	if h.ListOutputByCommand[command] && h.OutputStringByCommand[command] != "" {
		return fmt.Sprintf(h.OutputStringByCommand[command], args...)
	}
	return fmt.Sprintf(defaultTemplate, args...)
}

// FormatOutputString formats a string using output_string from commands.json if available, otherwise uses the provided template.
func (h *Handler) FormatOutputString(command string, template string, args ...interface{}) string {
	// If output_string is defined in JSON for this command, use it; otherwise use provided template
	outputTemplate := h.OutputStringByCommand[command]
	if outputTemplate == "" {
		outputTemplate = template
	}
	return fmt.Sprintf(outputTemplate, args...)
}

// SendWithFormat sends a message using the output_format defined in commands.json for the given command.
func (h *Handler) SendWithFormat(chatID int64, text string, command string) int {
	format := h.OutputFormatByCommand[command]
	if format != "markdown" && format != "plain" {
		format = "plain"
	}
	return h.SendMessage.Send(text, chatID, format == "markdown")
}

// StringReplacer interface for replacing strings
type StringReplacer interface {
	Replace(s string) string
}

// MessageSender interface for sending messages
type MessageSender interface {
	Send(text string, chatID int64, markdown bool) int
}

// Logger interface for logging
type Logger interface {
	Printf(format string, v ...interface{})
}

// ListHandler lists torrents, optionally filtered by tracker
func (h *Handler) List(ud tgbotapi.Update, tokens []string, cmd string) {
	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*list:* "+err.Error(), cmd)
		return
	}

	buf := new(bytes.Buffer)
	// if it gets a query, it will list torrents that has trackers that match the query
	if len(tokens) != 0 {
		// (?i) for case insensitivity
		regx, err := regexp.Compile("(?i)" + tokens[0])
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*list:* "+err.Error(), cmd)
			return
		}

		for i := range torrents {
			for _, tracker := range torrents[i].Trackers {
				if regx.MatchString(tracker.Announce) {
					buf.WriteString(h.FormatListLine(cmd, "<%d> %s\n", torrents[i].ID, torrents[i].Name))
					break
				}
			}
		}
	} else { // if we did not get a query, list all torrents
		for i := range torrents {
			buf.WriteString(h.FormatListLine(cmd, "<%d> %s\n", torrents[i].ID, torrents[i].Name))
		}
	}

	if buf.Len() == 0 {
		// if we got a tracker query show different message
		if len(tokens) != 0 {
			h.SendWithFormat(ud.Message.Chat.ID, "*list:* no matches", cmd)
			return
		}
		h.SendWithFormat(ud.Message.Chat.ID, "*list:* no torrents", cmd)
		return
	}

	h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)
}

// Head lists the first n torrents (default: 5)
func (h *Handler) Head(ud tgbotapi.Update, tokens []string, cmd string) {
	var (
		n   = 5 // default to 5
		err error
	)

	if len(tokens) > 0 {
		n, err = strconv.Atoi(tokens[0])
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*head:* "+err.Error(), cmd)
			return
		}
	}

	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*head:* "+err.Error(), cmd)
		return
	}

	// make sure that we stay in the boundaries
	if n <= 0 || n > len(torrents) {
		n = len(torrents)
	}

	buf := new(bytes.Buffer)
	for i := range torrents[:n] {
		torrentName := h.Replacer.Replace(torrents[i].Name) // escape markdown
		buf.WriteString(h.FormatListLine(cmd, "`<%d>` *%s*\n%s *%s* of *%s* (*%.1f%%*) ↓ *%s*  ↑ *%s* R: *%s*\n\n",
			torrents[i].ID, torrentName, torrents[i].TorrentStatus(), humanize.Bytes(torrents[i].Have()),
			humanize.Bytes(torrents[i].SizeWhenDone), torrents[i].PercentDone*100, humanize.Bytes(torrents[i].RateDownload),
			humanize.Bytes(torrents[i].RateUpload), torrents[i].Ratio()))
	}

	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*head:* no torrents", cmd)
		return
	}

	msgID := h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)

	if h.NoLive {
		return
	}

	// keep the info live
	for i := 0; i < h.Duration; i++ {
		time.Sleep(time.Second * h.Interval)
		buf.Reset()

		torrents, err = h.Client.GetTorrents()
		if err != nil {
			continue // try again if some error happened
		}

		if len(torrents) < 1 {
			break
		}

		// make sure that we stay in the boundaries
		if n <= 0 || n > len(torrents) {
			n = len(torrents)
		}

		for _, torrent := range torrents[:n] {
			torrentName := h.Replacer.Replace(torrent.Name) // escape markdown
			buf.WriteString(h.FormatListLine(cmd, "`<%d>` *%s*\n%s *%s* of *%s* (*%.1f%%*) ↓ *%s*  ↑ *%s* R: *%s*\n\n",
				torrent.ID, torrentName, torrent.TorrentStatus(), humanize.Bytes(torrent.Have()),
				humanize.Bytes(torrent.SizeWhenDone), torrent.PercentDone*100, humanize.Bytes(torrent.RateDownload),
				humanize.Bytes(torrent.RateUpload), torrent.Ratio()))
		}

		// no need to check if it is empty, as if the buffer is empty telegram won't change the message
		editConf := tgbotapi.NewEditMessageText(ud.Message.Chat.ID, msgID, buf.String())
		editConf.ParseMode = tgbotapi.ModeMarkdown
		h.Bot.Send(editConf)
	}
}

// Tail lists the last n torrents (default: 5)
func (h *Handler) Tail(ud tgbotapi.Update, tokens []string, cmd string) {
	var (
		n   = 5 // default to 5
		err error
	)

	if len(tokens) > 0 {
		n, err = strconv.Atoi(tokens[0])
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*tail:* "+err.Error(), cmd)
			return
		}
	}

	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*tail:* "+err.Error(), cmd)
		return
	}

	// make sure that we stay in the boundaries
	if n <= 0 || n > len(torrents) {
		n = len(torrents)
	}

	buf := new(bytes.Buffer)
	for _, torrent := range torrents[len(torrents)-n:] {
		torrentName := h.Replacer.Replace(torrent.Name) // escape markdown
		buf.WriteString(h.FormatListLine(cmd, "`<%d>` *%s*\n%s *%s* of *%s* (*%.1f%%*) ↓ *%s*  ↑ *%s* R: *%s*\n\n",
			torrent.ID, torrentName, torrent.TorrentStatus(), humanize.Bytes(torrent.Have()),
			humanize.Bytes(torrent.SizeWhenDone), torrent.PercentDone*100, humanize.Bytes(torrent.RateDownload),
			humanize.Bytes(torrent.RateUpload), torrent.Ratio()))
	}

	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*tail:* no torrents", cmd)
		return
	}

	msgID := h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)

	if h.NoLive {
		return
	}

	// keep the info live
	for i := 0; i < h.Duration; i++ {
		time.Sleep(time.Second * h.Interval)
		buf.Reset()

		torrents, err = h.Client.GetTorrents()
		if err != nil {
			continue // try again if some error happened
		}

		if len(torrents) < 1 {
			break
		}

		// make sure that we stay in the boundaries
		if n <= 0 || n > len(torrents) {
			n = len(torrents)
		}

		for _, torrent := range torrents[len(torrents)-n:] {
			torrentName := h.Replacer.Replace(torrent.Name) // escape markdown
			buf.WriteString(h.FormatListLine(cmd, "`<%d>` *%s*\n%s *%s* of *%s* (*%.1f%%*) ↓ *%s*  ↑ *%s* R: *%s*\n\n",
				torrent.ID, torrentName, torrent.TorrentStatus(), humanize.Bytes(torrent.Have()),
				humanize.Bytes(torrent.SizeWhenDone), torrent.PercentDone*100, humanize.Bytes(torrent.RateDownload),
				humanize.Bytes(torrent.RateUpload), torrent.Ratio()))
		}

		// no need to check if it is empty, as if the buffer is empty telegram won't change the message
		editConf := tgbotapi.NewEditMessageText(ud.Message.Chat.ID, msgID, buf.String())
		editConf.ParseMode = tgbotapi.ModeMarkdown
		h.Bot.Send(editConf)
	}
}

package handlers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pyed/transmission"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Handler contains the necessary clients and configuration for handling commands
type Handler struct {
	Bot                     *tgbotapi.BotAPI
	Client                  *transmission.TransmissionClient
	RPCURL                  string
	RPCUsername             string
	RPCPassword             string
	BotToken                string // used to download files from Telegram
	DefaultTorrentLocation  string // directory where received .torrent files are saved before adding to Transmission
	DefaultDownloadLocation string // directory where downloaded files are stored
	NoLive                  bool
	DefaultMoveLocation     string // directory where completed downloads should be copied/moved to

	Interval              time.Duration
	Duration              int
	StartTime             time.Time
	UpdateMaxIterations   int // max live-update iterations per message (0 = disable live updates)
	Replacer              StringReplacer
	SendMessage           MessageSender
	Logger                Logger
	OutputFormatByCommand map[string]string // canonical command name -> "markdown" or "plain"
	OutputStringByCommand map[string]string // canonical command name -> format string for output
	ListOutputByCommand   map[string]bool   // when true, output_string is used to format each line of list output
	CompletedFilePath     string

	liveMu      sync.Mutex
	liveCancels map[string]liveTask
	liveSeq     int64
}

type liveTask struct {
	seq    int64
	cancel context.CancelFunc
}

// StartLiveTask starts a cancellable task for a key and cancels any prior task with the same key.
func (h *Handler) StartLiveTask(key string) (context.Context, func()) {
	h.liveMu.Lock()
	if h.liveCancels == nil {
		h.liveCancels = make(map[string]liveTask)
	}
	if prev, ok := h.liveCancels[key]; ok {
		prev.cancel()
	}
	h.liveSeq++
	seq := h.liveSeq
	ctx, cancel := context.WithCancel(context.Background())
	h.liveCancels[key] = liveTask{seq: seq, cancel: cancel}
	h.liveMu.Unlock()

	finish := func() {
		h.liveMu.Lock()
		if current, ok := h.liveCancels[key]; ok && current.seq == seq {
			delete(h.liveCancels, key)
		}
		h.liveMu.Unlock()
		cancel()
	}

	return ctx, finish
}

// FormatOutputString formats a string using output_string from commands.json if available, otherwise uses "(RAW) %s".
func (h *Handler) FormatOutputString(command string, args ...interface{}) string {
	cmdKey := strings.ToLower(strings.TrimSpace(command))
	outputTemplate := h.OutputStringByCommand[cmdKey]
	if outputTemplate == "" {
		outputTemplate = "(RAW) %s"
	}
	return fmt.Sprintf(outputTemplate, args...)
}

// SendWithFormat sends a message using the output_format defined in commands.json for the given command.
func (h *Handler) SendWithFormat(chatID int64, text string, command string, args ...interface{}) int {
	cmdKey := strings.ToLower(strings.TrimSpace(command))
	format := strings.ToLower(strings.TrimSpace(h.OutputFormatByCommand[cmdKey]))

	if format != "markdown" && format != "plain" {
		format = "plain" // default to plain if not specified or invalid
	}

	if len(args) > 0 {
		if override, ok := args[len(args)-1].(string); ok {
			normalized := strings.ToLower(strings.TrimSpace(override))
			if normalized == "markdown" || normalized == "plain" {
				format = normalized
			}
		}
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

// if it gets a query, it will list torrents that has trackers that match the query

// (?i) for case insensitivity

// if we did not get a query, list all torrents

// if we got a tracker query show different message

// Head lists the first n torrents (default: 5)

// default to 5

// make sure that we stay in the boundaries

// escape markdown

// Tail lists the last n torrents (default: 5)

// default to 5

// make sure that we stay in the boundaries

// escape markdown

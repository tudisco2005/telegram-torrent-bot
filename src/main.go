package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pyed/transmission"
	"github.com/tudisco2005/telegram-torrent-bot/config"
	"github.com/tudisco2005/telegram-torrent-bot/telegram"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

const (
	VERSION = "v1.0.0"
)

// AppConfig holds the application configuration and clients
type AppConfig struct {
	BotToken     string
	Masters      config.MasterSlice
	RPCURL       string
	Username     string
	Password     string
	LogFile      string
	TransLogFile string // Transmission log file
	NoLive       bool

	// transmission
	Client *transmission.TransmissionClient

	// telegram
	Bot     *tgbotapi.BotAPI
	Updates <-chan tgbotapi.Update

	// chatID will be used to keep track of which chat to send completion notifications.
	ChatID int64

	// logging
	Logger *log.Logger

	// interval in seconds for live updates, affects: "active", "info", "speed", "head", "tail"
	Interval time.Duration
	// duration controls how many intervals will happen
	Duration int

	// verbose mode for debug logging
	Verbose bool
}

func main() {
	// Initialize configuration
	cfg := &AppConfig{
		Logger:   log.New(os.Stdout, "", log.LstdFlags),
		Interval: 5,
		Duration: 10,
		Masters:  config.MasterSlice{},
	}

	// Parse flags and initialize
	if err := InitFlags(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize flags: %v\n", err)
		os.Exit(1)
	}

	// Initialize transmission client
	if err := InitTransmission(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize transmission: %v\n", err)
		os.Exit(1)
	}

	// Initialize telegram bot
	if err := InitTelegram(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize telegram: %v\n", err)
		os.Exit(1)
	}

	// Setup message sender
	botCfg := &telegram.BotConfig{
		Bot:          cfg.Bot,
		Updates:      cfg.Updates,
		Masters:      cfg.Masters,
		Client:       cfg.Client,
		NoLive:       cfg.NoLive,
		Interval:     cfg.Interval,
		Duration:     cfg.Duration,
		Logger:       cfg.Logger,
		SendMessage:  &telegram.SimpleMessageSender{Bot: cfg.Bot, Logger: cfg.Logger, Verbose: cfg.Verbose},
		ChatID:       cfg.ChatID,
		TransLogFile: cfg.TransLogFile,
		VERSION:      VERSION,
		Verbose:      cfg.Verbose,
	}

	// Start Telegram bot event loop
	telegram.Start(botCfg)
}

// sendMessage sends a message to telegram with optional markdown
func sendMessage(bot *tgbotapi.BotAPI, text string, chatID int64, markdown bool) int {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.DisableWebPagePreview = true
	if markdown {
		msg.ParseMode = tgbotapi.ModeMarkdown
	}

	resp, err := bot.Send(msg)
	if err != nil {
		return 0
	}

	return resp.MessageID
}

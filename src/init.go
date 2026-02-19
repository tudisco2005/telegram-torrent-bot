package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pyed/transmission"
	"github.com/tudisco2005/telegram-torrent-bot/config"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// InitFlags parses and validates command line flags
func InitFlags(cfg *AppConfig) error {
	// define arguments and parse them.
	flag.StringVar(&cfg.BotToken, "token", "", "Telegram bot token, Can be passed via environment variable 'TT_BOTT'")
	flag.Var(&cfg.Masters, "master", "Your telegram handler, So the bot will only respond to you. Can specify more than one")
	flag.StringVar(&cfg.RPCURL, "url", "http://localhost:9091/transmission/rpc", "Transmission RPC URL")
	flag.StringVar(&cfg.Username, "username", "", "Transmission username")
	flag.StringVar(&cfg.Password, "password", "", "Transmission password")
	flag.StringVar(&cfg.LogFile, "logfile", "", "Send logs to a file")
	flag.BoolVar(&cfg.NoLive, "no-live", false, "Don't edit and update info after sending")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Enable verbose debug logging (prints all received messages and more)")

	// set the usage message
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, "Usage: transmission-telegram <-token=TOKEN> <-master=@tuser> [-master=@yuser2] [-url=http://] [-username=user] [-password=pass]\n\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Load environment configuration from .env file and environment variables
	envCfg := &config.EnvConfig{
		BotToken:                &cfg.BotToken,
		Masters:                 &cfg.Masters,
		RPCURL:                  &cfg.RPCURL,
		Username:                &cfg.Username,
		Password:                &cfg.Password,
		LogFile:                 &cfg.LogFile,
		DefaultTorrentLocation:  &cfg.DefaultTorrentLocation,
		DefaultDownloadLocation: &cfg.DefaultDownloadLocation,
		NoLive:                  &cfg.NoLive,
		Verbose:                 &cfg.Verbose,
		UpdateMaxIterations:     &cfg.UpdateMaxIterations,
	}
	config.LoadEnvironmentConfig(envCfg)

	// make sure that we have the two mandatory arguments: telegram token & master's handler.
	if cfg.BotToken == "" || len(cfg.Masters) < 1 {
		fmt.Fprintf(os.Stderr, "Error: Mandatory argument missing! (-token or -master)\n\n")
		flag.Usage()
		return fmt.Errorf("missing mandatory arguments")
	}

	// make sure that the handler doesn't contain @ and convert to lowercase
	for i := range cfg.Masters {
		cfg.Masters[i] = strings.Replace(cfg.Masters[i], "@", "", -1)
		cfg.Masters[i] = strings.ToLower(cfg.Masters[i])
	}

	// if we got a log file, log to it
	if cfg.LogFile != "" {
		logf, err := os.OpenFile(cfg.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return err
		}

		// Crea un MultiWriter che scrive sia su stdout che sul file
		multiWriter := io.MultiWriter(os.Stdout, logf)

		// Imposta il MultiWriter come output del logger
		cfg.Logger.SetOutput(multiWriter)
	}

	// log the flags
	cfg.Logger.Printf("[INFO] Token=%s\n\t\tMasters=%s\n\t\tURL=%s\n\t\tUSER=%s\n\t\tPASS=%s",
		cfg.BotToken, cfg.Masters, cfg.RPCURL, cfg.Username, cfg.Password)

	return nil
}

// InitTransmission initializes the transmission client
func InitTransmission(cfg *AppConfig) error {
	var err error
	cfg.Client, err = transmission.New(cfg.RPCURL, cfg.Username, cfg.Password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Transmission: Make sure you have the right URL, Username and Password\n")
		return err
	}
	return nil
}

// InitTelegram initializes the telegram bot
func InitTelegram(cfg *AppConfig) error {
	// authorize using the token
	var err error
	cfg.Bot, err = tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Telegram: %s\n", err)
		return err
	}
	cfg.Logger.Printf("[INFO] Authorized: %s", cfg.Bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	cfg.Updates, err = cfg.Bot.GetUpdatesChan(u)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Telegram: %s\n", err)
		return err
	}

	return nil
}

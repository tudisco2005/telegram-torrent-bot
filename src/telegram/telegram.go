package telegram

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/pyed/transmission"
	"github.com/tudisco2005/telegram-torrent-bot/config"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	"github.com/tudisco2005/telegram-torrent-bot/utils"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Command represents a command definition from JSON
type Command struct {
	Name            string   `json:"name"`
	CommandCategory string   `json:"command_category"`
	Aliases         []string `json:"aliases"`
	Description     string   `json:"description"`
	Example         string   `json:"example,omitempty"`
	OutputFormat    string   `json:"output_format,omitempty"` // "markdown" or "plain"
	OutputString    string   `json:"output_string,omitempty"` // Format string for command output (uses fmt.Sprintf placeholders)
	ListOutput      bool     `json:"list_output,omitempty"`   // when true, output_string formats each line of a list
}

// Commands holds all command definitions
type Commands struct {
	Commands []Command `json:"commands"`
}

// Category represents a command category definition from JSON
type Category struct {
	ID    string `json:"id"`
	Emoji string `json:"emoji"`
	Name  string `json:"name"`
}

// Categories holds all category definitions
type Categories struct {
	Categories []Category `json:"categories"`
}

// LoadCommands loads command definitions from JSON file
func LoadCommands(path string) (*Commands, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cmds Commands
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cmds); err != nil {
		return nil, err
	}
	return &cmds, nil
}

// LoadCategories loads category definitions from JSON file
func LoadCategories(path string) (*Categories, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cats Categories
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cats); err != nil {
		return nil, err
	}
	return &cats, nil
}

// BotConfig holds telegram bot configuration
type BotConfig struct {
	Bot                     *tgbotapi.BotAPI
	Updates                 <-chan tgbotapi.Update
	Masters                 config.MasterSlice
	Client                  *transmission.TransmissionClient
	NoLive                  bool
	Interval                time.Duration
	Duration                int
	UpdateMaxIterations     int // max live-update iterations per message (0 = use Duration)
	Logger                  *log.Logger
	SendMessage             MessageSender
	ChatID                  int64
	DefaultTorrentLocation  string // directory where received .torrent files are saved before adding to Transmission
	DefaultDownloadLocation string // directory where downloaded files are stored
	VERSION                 string
	Verbose                 bool
}

// MessageSender interface for sending messages
type MessageSender interface {
	Send(text string, chatID int64, markdown bool) int
}

// SimpleMessageSender implements MessageSender
type SimpleMessageSender struct {
	Bot     *tgbotapi.BotAPI
	Logger  *log.Logger
	Verbose bool
}

// Send sends a message to telegram
func (s *SimpleMessageSender) Send(text string, chatID int64, markdown bool) int {
	if s.Verbose {
		textPreview := text
		if len(textPreview) > 100 {
			textPreview = textPreview[:100] + "..."
		}
		s.Logger.Printf("[DEBUG] Sending message to ChatID=%d, Markdown=%v, Length=%d, Preview=%q",
			chatID, markdown, len(text), textPreview)
	}
	return sendMessage(s.Bot, text, chatID, markdown)
}

// sendMessage helper function
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

// Start begins the telegram bot event loop
func Start(cfg *BotConfig) {
	// Load commands from JSON
	cmds, err := LoadCommands("telegram/commands.json")
	if err != nil {
		cfg.Logger.Printf("Warning: Failed to load commands from JSON: %v", err)
		cmds = &Commands{Commands: []Command{}} // Use empty commands list as fallback
	}

	// Build output format map from commands (canonical name -> "markdown" or "plain")
	outputFormatByCommand := make(map[string]string)
	// Build output string map from commands (canonical name -> format string)
	outputStringByCommand := make(map[string]string)
	// Build list_output map (canonical name -> use output_string per line when true)
	listOutputByCommand := make(map[string]bool)
	for _, cmd := range cmds.Commands {
		format := cmd.OutputFormat
		if format != "markdown" && format != "plain" {
			format = "plain"
		}
		outputFormatByCommand[cmd.Name] = format
		if cmd.OutputString != "" {
			outputStringByCommand[cmd.Name] = cmd.OutputString
		}
		listOutputByCommand[cmd.Name] = cmd.ListOutput
	}

	// Create handler
	h := &handlers.Handler{
		Bot:                     cfg.Bot,
		Client:                  cfg.Client,
		BotToken:                cfg.Bot.Token,
		DefaultTorrentLocation:  cfg.DefaultTorrentLocation,
		DefaultDownloadLocation: cfg.DefaultDownloadLocation,
		NoLive:                  cfg.NoLive,
		Interval:                cfg.Interval * time.Second,
		Duration:                cfg.Duration,
		UpdateMaxIterations:     cfg.UpdateMaxIterations,
		Replacer:                utils.MarkdownReplacer,
		SendMessage:             cfg.SendMessage,
		Logger:                  cfg.Logger,
		OutputFormatByCommand:   outputFormatByCommand,
		OutputStringByCommand:   outputStringByCommand,
		ListOutputByCommand:     listOutputByCommand,
	}

	cfg.Logger.Printf("[DEBUG] Bot started with version %s", cfg.VERSION)

	// read chat.json to get chat ID for sending startup message (it can be an array)
	chatIDs, err := utils.LoadChatIDs("telegram/chat.json")
	if err != nil || len(chatIDs) == 0 {
		cfg.Logger.Printf("Warning: Failed to load chat IDs from JSON: %v", err)
		cfg.ChatID = 0 // fallback to 0, which will cause sendMessage to fail gracefully
	}

	// Send startup message to all chat IDs loaded from JSON (if any)
	startupMsg := fmt.Sprintf("*Bot Online!*\nVersion: %s\n\nSend 'help' for list of commands.", cfg.VERSION)
	for _, chatID := range chatIDs {
		cfg.SendMessage.Send(startupMsg, chatID, true)
	}

	cfg.Logger.Printf("[DEBUG] Startup message sent to %d chat(s)", len(chatIDs))

	// Main event loop
	for update := range cfg.Updates {
		// Verbose logging: log all updates received
		if cfg.Verbose {
			cfg.Logger.Printf("[DEBUG] Update received: UpdateID=%d", update.UpdateID)
			if update.Message != nil {
				cfg.Logger.Printf("[DEBUG] Message: ChatID=%d, From=%s (ID:%d), Text=%q, Date=%d",
					update.Message.Chat.ID,
					update.Message.From.UserName,
					update.Message.From.ID,
					update.Message.Text,
					update.Message.Date)
				if update.Message.Document != nil {
					cfg.Logger.Printf("[DEBUG] Document: FileID=%s, FileName=%s, FileSize=%d",
						update.Message.Document.FileID,
						update.Message.Document.FileName,
						update.Message.Document.FileSize)
				}
			} else {
				cfg.Logger.Printf("[DEBUG] Update without message (edited message or other type)")
			}
		}

		// ignore edited messages
		if update.Message == nil {
			if cfg.Verbose {
				cfg.Logger.Printf("[DEBUG] Ignoring update: no message")
			}
			continue
		}

		// ignore non masters
		if !cfg.Masters.Contains(update.Message.From.UserName) {
			if cfg.Verbose {
				cfg.Logger.Printf("[DEBUG] Ignoring message: user %s is not a master (masters: %v)",
					update.Message.From.UserName, cfg.Masters)
			}
			continue
		}

		if cfg.Verbose {
			cfg.Logger.Printf("[DEBUG] User %s is authorized master", update.Message.From.UserName)
		}

		// Skip empty messages
		if update.Message.Text == "" && update.Message.Document == nil {
			if cfg.Verbose {
				cfg.Logger.Printf("[DEBUG] Skipping empty message")
			}
			continue
		}

		// tokenize the update
		tokens := strings.Split(update.Message.Text, " ")

		// Extract command and remove '/' prefix if present
		command := strings.TrimPrefix(strings.ToLower(tokens[0]), "/")
		var args []string
		if len(tokens) > 1 {
			args = tokens[1:]
		}

		// If message is a bare URL/magnet (no leading /add), treat it as an add command
		if strings.HasPrefix(tokens[0], "magnet") || strings.HasPrefix(tokens[0], "http") {
			if cfg.Verbose {
				cfg.Logger.Printf("[DEBUG] Detected URL/magnet link, prepending 'add' command")
			}
			command = "add"
			args = []string{update.Message.Text}
		}

		// If this is an add command and the first arg looks like a URL/magnet,
		// join all remaining tokens into a single argument to avoid splitting on spaces
		if command == "add" && len(args) > 0 && (strings.HasPrefix(args[0], "magnet") || strings.HasPrefix(args[0], "http")) {
			if cfg.Verbose {
				cfg.Logger.Printf("[DEBUG] Detected add with split URL, joining args into single URL")
			}
			args = []string{strings.Join(args, " ")}
		}

		if cfg.Verbose {
			cfg.Logger.Printf("[DEBUG] Processing command: %s, args: %v", command, args)
		}

		// Dispatch command
		dispatchCommand(h, cfg, cmds, update, command, args)
	}
}

// generateHelpMessage creates a formatted help message from commands
func generateHelpMessage(cmds *Commands) string {
	var buf strings.Builder
	buf.WriteString("📋 *Available Commands*\n\n")

	// Load categories from JSON
	cats, err := LoadCategories("telegram/categories.json")
	if err != nil {
		// Fallback: return a simple help message if categories file is not found
		buf.WriteString("_Error loading command categories. Please try again later._")
		return buf.String()
	}

	// Build maps from categories
	categoryEmojis := make(map[string]string)
	categoryNames := make(map[string]string)
	categoryOrder := []string{}

	for _, cat := range cats.Categories {
		categoryEmojis[cat.ID] = cat.Emoji
		categoryNames[cat.ID] = cat.Name
		categoryOrder = append(categoryOrder, cat.ID)
	}

	// Group commands by category
	categoryMap := make(map[string][]Command)
	for _, cmd := range cmds.Commands {
		category := cmd.CommandCategory
		categoryMap[category] = append(categoryMap[category], cmd)
	}

	// Generate help for each category
	for _, category := range categoryOrder {
		commands, exists := categoryMap[category]
		if !exists || len(commands) == 0 {
			continue
		}

		// Write category header
		emoji := categoryEmojis[category]
		name := categoryNames[category]
		buf.WriteString("*" + emoji + " " + name + "*\n")

		// Write commands in this category
		for _, cmd := range commands {
			// Build command line: - `/command` or `/alias1` or `/alias2` - description
			cmdLine := "- `/" + cmd.Name + "`"
			if len(cmd.Aliases) > 0 {
				for _, alias := range cmd.Aliases {
					cmdLine += " or `/" + alias + "`"
				}
			}
			cmdLine += " - " + cmd.Description
			buf.WriteString(cmdLine + "\n")

			// Add example if available
			if cmd.Example != "" {
				buf.WriteString("  _Example:_ `" + cmd.Example + "`\n")
			}
		}

		buf.WriteString("\n")
	}

	// Add footer
	buf.WriteString("---\n\n")
	buf.WriteString("💡 *Tips:*\n")
	buf.WriteString("- Prefix commands with `/` if you want to use the bot in a group\n")
	buf.WriteString("- Report any issues [here](https://github.com/tudisco2005/telegram-torrent-bot)")

	return buf.String()
}

// getCanonicalName returns the canonical command name (from JSON) for a given input (name or alias).
func getCanonicalName(cmds *Commands, input string) string {
	input = strings.ToLower(input)
	for _, cmd := range cmds.Commands {
		if strings.ToLower(cmd.Name) == input {
			return cmd.Name
		}
		for _, alias := range cmd.Aliases {
			if strings.ToLower(alias) == input {
				return cmd.Name
			}
		}
	}
	return ""
}

// CommandHandler represents a function that handles a command (canonical = command name from JSON).
type CommandHandler func(*handlers.Handler, tgbotapi.Update, []string, string)

// buildCommandMap creates a map from command/alias names to handler functions
func buildCommandMap(cmds *Commands) map[string]CommandHandler {
	cmdMap := make(map[string]CommandHandler)

	// Map command names to their handler functions; handlers receive canonical command name for output_format
	handlerMap := map[string]CommandHandler{
		"list":        func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.List(u, a, c) },
		"head":        func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Head(u, a, c) },
		"tail":        func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Tail(u, a, c) },
		"downs":       func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Downs(u, c) },
		"seeding":     func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Seeding(u, c) },
		"paused":      func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Paused(u, c) },
		"checking":    func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Checking(u, c) },
		"active":      func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Active(u, c) },
		"errors":      func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Errors(u, c) },
		"sort":        func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Sort(u, a, c) },
		"trackers":    func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Trackers(u, c) },
		"downloaddir": func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.DownloadDir(u, a, c) },
		"add":         func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Add(u, a, c) },
		"search":      func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Search(u, a, c) },
		"latest":      func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Latest(u, a, c) },
		"info":        func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Info(u, a, c) },
		"stop":        func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Stop(u, a, c) },
		"start":       func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Start(u, a, c) },
		"check":       func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Check(u, a, c) },
		"del":         func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Delete(u, a, c) },
		"deldata":     func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.DeleteData(u, a, c) },
		"stats":       func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Stats(u, c) },
		"downlimit":   func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.DownloadLimit(u, a, c) },
		"uplimit":     func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.UploadLimit(u, a, c) },
		"speed":       func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Speed(u, c) },
		"count":       func(h *handlers.Handler, u tgbotapi.Update, a []string, c string) { h.Count(u, c) },
		// Note: "version" and "help" are handled separately in dispatchCommand
	}

	// Build map from JSON commands: map each command name and alias to its handler
	for _, cmd := range cmds.Commands {
		if handler, exists := handlerMap[cmd.Name]; exists {
			cmdMap[strings.ToLower(cmd.Name)] = handler
			for _, alias := range cmd.Aliases {
				cmdMap[strings.ToLower(alias)] = handler
			}
		}
	}

	return cmdMap
}

// dispatchCommand routes commands to appropriate handlers using the command map from JSON
func dispatchCommand(h *handlers.Handler, cfg *BotConfig, cmds *Commands, update tgbotapi.Update, command string, args []string) {
	// Build command map from JSON (cache could be added here for performance)
	cmdMap := buildCommandMap(cmds)

	if cfg.Verbose {
		cfg.Logger.Printf("[DEBUG] Dispatching command: %s", command)
	}

	// Look up command in map and dispatch with canonical name for output_format
	canonical := getCanonicalName(cmds, command)

	// Handle special case for version command (needs VERSION string)
	if canonical == "version" {
		if cfg.Verbose {
			cfg.Logger.Printf("[DEBUG] Executing version command")
		}
		h.Version(update, cfg.VERSION)
		return
	}

	// Handle help command (use output_format from JSON), read alias from JSON
	if canonical == "help" {
		if cfg.Verbose {
			cfg.Logger.Printf("[DEBUG] Executing help command")
		}
		helpMsg := generateHelpMessage(cmds)
		useMarkdown := h.OutputFormatByCommand["help"] == "markdown"
		cfg.SendMessage.Send(helpMsg, update.Message.Chat.ID, useMarkdown)
		return
	}

	if handler, exists := cmdMap[command]; exists && canonical != "" {
		if cfg.Verbose {
			cfg.Logger.Printf("[DEBUG] Found handler for command: %s (canonical: %s)", command, canonical)
		}

		handler(h, update, args, canonical)
		return
	}

	// Default: Check if it's a torrent file
	if update.Message.Document != nil {
		if cfg.Verbose {
			cfg.Logger.Printf("[DEBUG] No command match, treating as torrent file")
		}
		h.ReceiveTorrent(update)
	} else {
		if cfg.Verbose {
			cfg.Logger.Printf("[DEBUG] Unknown command: %s, showing help", command)
		}
		// send command not found
		cfg.SendMessage.Send(fmt.Sprintf("Unknown command: %s, send 'help' for list all commands", command), update.Message.Chat.ID, false)

		// no such command, show help (use output_format for "help" from JSON)
		// helpMsg := generateHelpMessage(cmds)
		// useMarkdown := h.OutputFormatByCommand["help"] == "markdown"
		// cfg.SendMessage.Send(helpMsg, update.Message.Chat.ID, useMarkdown)
	}
}

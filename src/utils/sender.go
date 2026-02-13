package utils

import (
	"strings"
	"unicode/utf8"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Sender interface for sending messages
type Sender interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
}

// SendMessage sends a message to a chat with optional markdown formatting
// It handles message splitting for messages longer than 4096 characters
func SendMessage(sender Sender, text string, chatID int64, markdown bool) int {
	// set typing action
	action := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	sender.Send(action)

	// check the rune count, telegram is limited to 4096 chars per message;
	// so if our message is > 4096, split it in chunks then send them.
	msgRuneCount := utf8.RuneCountInString(text)

LenCheck:
	if msgRuneCount > 4096 {
		// find the last newline within the 4096 char limit
		runes := []rune(text)
		chunk := string(runes[:4096])
		lastNewline := strings.LastIndex(chunk, "\n")

		if lastNewline == -1 {
			lastNewline = 4096
		}

		// send current chunk
		msg := tgbotapi.NewMessage(chatID, string(runes[:lastNewline]))
		msg.DisableWebPagePreview = true
		if markdown {
			msg.ParseMode = tgbotapi.ModeMarkdown
		}
		sender.Send(msg)

		// move to the next chunk
		text = string(runes[lastNewline:])
		msgRuneCount = utf8.RuneCountInString(text)
		goto LenCheck
	}

	// if msgRuneCount < 4096, send it normally
	msg := tgbotapi.NewMessage(chatID, text)
	msg.DisableWebPagePreview = true
	if markdown {
		msg.ParseMode = tgbotapi.ModeMarkdown
	}

	resp, err := sender.Send(msg)
	if err != nil {
		return 0
	}

	return resp.MessageID
}

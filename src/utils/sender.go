package utils

import (
	"encoding/json"
	"os"
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

// ChatIDs holds the list of chat IDs loaded from JSON
type ChatID struct {
	ChatID int64 `json:"chat_id"`
	Timestamp int64 `json:"timestamp"`
}

type ChatIDs struct {
	ChatIDs []int64 `json:"chat_ids"`
}

func LoadChatIDs(path string) (ChatIDs, error) {
	var chatIDs ChatIDs
	file, err := os.Open(path)
	if err != nil {
		return chatIDs, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&chatIDs); err != nil {
		return ChatIDs{}, err
	}

	return chatIDs, nil
}

func SaveChatIDs(chatIDs ChatIDs) string {
	// load existing chat IDs
	existingChatIDs, err := LoadChatIDs("telegram/chat.json")
	if err != nil {
		existingChatIDs = ChatIDs{ChatIDs: []int64{}}
	}

	// merge existing and new chat IDs
	mergedChatIDs := existingChatIDs.ChatIDs
	for _, newID := range chatIDs.ChatIDs {
		found := false
		for _, existingID := range mergedChatIDs {
			if existingID == newID {
				found = true
				break
			}
		}
		if !found {
			mergedChatIDs = append(mergedChatIDs, newID)
		}
	}

	// remove the chat ID older than 30 days
	oneMonthAgo := int64(30 * 24 * 60 * 60)
	now := int64(0)
	for _, chatID := range mergedChatIDs {
		chatIDObj, err := LoadChatID(chatID)
		if err != nil {
			continue
		}
		if chatIDObj.Timestamp < now-oneMonthAgo {
			// remove this chat ID from mergedChatIDs
			for i, id := range mergedChatIDs {
				if id == chatID {
					mergedChatIDs = append(mergedChatIDs[:i], mergedChatIDs[i+1:]...)
					break
				}
			}
		}
	}

	// save merged chat IDs back to file
	file, err := os.OpenFile("telegram/chat.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err.Error()
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(ChatIDs{ChatIDs: mergedChatIDs}); err != nil {
		return err.Error()
	}

	return ""
}
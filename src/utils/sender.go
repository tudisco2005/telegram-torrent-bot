package utils

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	ID        int64 `json:"id"`
	Timestamp int64 `json:"timestamp"`
}

// LoadChatIDs loads an array of ChatID objects from the given JSON file.
// If the file does not exist, it returns an empty slice and nil error.
func LoadChatIDs(path string) ([]ChatID, error) {
	var chatIDs []ChatID
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []ChatID{}, nil
		}
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&chatIDs); err != nil {
		return nil, err
	}

	return chatIDs, nil
}

// SaveChatIDs writes the provided ChatID slice to the given path as pretty JSON.
func SaveChatIDs(path string, chatIDs []ChatID) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(chatIDs); err != nil {
		return err
	}
	return nil
}

// GetIDs extracts just the int64 IDs from ChatID objects.
func GetIDs(chatIDs []ChatID) []int64 {
	ids := make([]int64, 0, len(chatIDs))
	for _, c := range chatIDs {
		ids = append(ids, c.ID)
	}
	return ids
}

// AddOrUpdateChatID ensures the given chat ID exists in the file, adding it with the current timestamp
// if missing or updating its timestamp if present. It does NOT prune old entries; pruning is
// performed once at bot startup by the caller. Returns (added, error) where added=true when
// a new ChatID was appended (not just updated).
func AddOrUpdateChatID(path string, id int64) (bool, error) {
	chatIDs, err := LoadChatIDs(path)
	if err != nil {
		return false, err
	}

	now := time.Now().Unix()
	found := false
	for i := range chatIDs {
		if chatIDs[i].ID == id {
			chatIDs[i].Timestamp = now
			found = true
			break
		}
	}
	added := false
	if !found {
		chatIDs = append(chatIDs, ChatID{ID: id, Timestamp: now})
		added = true
	}

	if err := SaveChatIDs(path, chatIDs); err != nil {
		return false, err
	}

	return added, nil
}

// LoadTracked loads an array of tracked torrent IDs from the given JSON file.
// If the file does not exist, it returns an empty slice and nil error.
func LoadTracked(path string) ([]int, error) {
	var ids []int
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []int{}, nil
		}
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&ids); err != nil {
		if err == io.EOF {
			return []int{}, nil
		}
		return nil, err
	}
	return ids, nil
}

// SaveTracked writes the provided slice of tracked torrent IDs to the given path as pretty JSON.
func SaveTracked(path string, ids []int) error {
	return writeJSONAtomic(path, ids)
}

func writeJSONAtomic(path string, v interface{}) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".tmp-json-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpPath)
	}()

	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(v); err != nil {
		return err
	}

	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Chmod(tmpPath, 0644); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}

	return nil
}

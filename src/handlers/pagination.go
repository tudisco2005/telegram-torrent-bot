package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

const (
	paginationCallbackPrefix = "pg:"
	paginationMaxRunes       = 750
	paginationTTL            = 5 * time.Minute
)

type paginationSession struct {
	Pages    []string
	Markdown bool
	Created  time.Time
}

func (h *Handler) SendWithPagination(chatID int64, text string, markdown bool) (int, bool) {
	pages := splitPages(text, paginationMaxRunes)
	if len(pages) <= 1 {
		return h.SendMessage.Send(text, chatID, markdown), false
	}

	msg := tgbotapi.NewMessage(chatID, formatPage(pages, 0))
	msg.DisableWebPagePreview = true
	if markdown {
		msg.ParseMode = tgbotapi.ModeMarkdown
	}
	keyboard := buildPaginationKeyboard(0, len(pages))
	msg.ReplyMarkup = &keyboard

	resp, err := h.Bot.Send(msg)
	if err != nil {
		h.Logger.Printf("[DEBUG] Failed to send paged message: chat=%d err=%v", chatID, err)
		return 0, false
	}

	h.paginationMu.Lock()
	if h.paginationSession == nil {
		h.paginationSession = make(map[string]paginationSession)
	}
	h.prunePaginationLocked(time.Now())
	h.paginationSession[paginationKey(chatID, resp.MessageID)] = paginationSession{
		Pages:    pages,
		Markdown: markdown,
		Created:  time.Now(),
	}
	h.paginationMu.Unlock()

	return resp.MessageID, true
}

func (h *Handler) SendWithPaginationFormat(chatID int64, text string, command string, args ...interface{}) (int, bool) {
	cmdKey := strings.ToLower(strings.TrimSpace(command))
	format := strings.ToLower(strings.TrimSpace(h.OutputFormatByCommand[cmdKey]))
	if format != "markdown" && format != "plain" {
		format = "plain"
	}

	if len(args) > 0 {
		if override, ok := args[len(args)-1].(string); ok {
			normalized := strings.ToLower(strings.TrimSpace(override))
			if normalized == "markdown" || normalized == "plain" {
				format = normalized
			}
		}
	}

	return h.SendWithPagination(chatID, text, format == "markdown")
}

func (h *Handler) HandlePaginationCallback(update tgbotapi.Update) bool {
	if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
		return false
	}

	data := update.CallbackQuery.Data
	if !strings.HasPrefix(data, paginationCallbackPrefix) {
		return false
	}

	pageText := strings.TrimPrefix(data, paginationCallbackPrefix)
	pageIndex, err := strconv.Atoi(pageText)
	if err != nil {
		h.answerCallback(update.CallbackQuery.ID, "Invalid page")
		return true
	}

	chatID := update.CallbackQuery.Message.Chat.ID
	msgID := update.CallbackQuery.Message.MessageID
	key := paginationKey(chatID, msgID)

	h.paginationMu.Lock()
	h.prunePaginationLocked(time.Now())
	session, ok := h.paginationSession[key]
	h.paginationMu.Unlock()

	if !ok || len(session.Pages) == 0 {
		h.answerCallback(update.CallbackQuery.ID, "This pagination expired")
		return true
	}

	if pageIndex < 0 {
		pageIndex = 0
	}
	if pageIndex >= len(session.Pages) {
		pageIndex = len(session.Pages) - 1
	}

	edit := tgbotapi.NewEditMessageText(chatID, msgID, formatPage(session.Pages, pageIndex))
	if session.Markdown {
		edit.ParseMode = tgbotapi.ModeMarkdown
	}
	keyboard := buildPaginationKeyboard(pageIndex, len(session.Pages))
	edit.ReplyMarkup = &keyboard

	if _, err := h.Bot.Send(edit); err != nil {
		h.Logger.Printf("[DEBUG] Failed to edit paged message: chat=%d msg=%d err=%v", chatID, msgID, err)
		h.answerCallback(update.CallbackQuery.ID, "Unable to switch page")
		return true
	}

	h.answerCallback(update.CallbackQuery.ID, fmt.Sprintf("Page %d/%d", pageIndex+1, len(session.Pages)))
	return true
}

func (h *Handler) answerCallback(callbackID string, text string) {
	answer := tgbotapi.NewCallback(callbackID, text)
	if _, err := h.Bot.AnswerCallbackQuery(answer); err != nil {
		h.Logger.Printf("[DEBUG] Failed to answer callback: id=%s err=%v", callbackID, err)
	}
}

func (h *Handler) prunePaginationLocked(now time.Time) {
	for key, session := range h.paginationSession {
		if now.Sub(session.Created) > paginationTTL {
			delete(h.paginationSession, key)
		}
	}
}

func paginationKey(chatID int64, messageID int) string {
	return fmt.Sprintf("%d:%d", chatID, messageID)
}

func formatPage(pages []string, pageIndex int) string {
	total := len(pages)
	if total == 0 {
		return ""
	}
	if pageIndex < 0 {
		pageIndex = 0
	}
	if pageIndex >= total {
		pageIndex = total - 1
	}

	return fmt.Sprintf("%s\n\nPage %d/%d", pages[pageIndex], pageIndex+1, total)
}

func buildPaginationKeyboard(pageIndex int, total int) tgbotapi.InlineKeyboardMarkup {
	buttons := []tgbotapi.InlineKeyboardButton{}

	if pageIndex > 0 {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("◀️ Prev", fmt.Sprintf("%s%d", paginationCallbackPrefix, pageIndex-1)))
	}
	if pageIndex < total-1 {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("Next ▶️", fmt.Sprintf("%s%d", paginationCallbackPrefix, pageIndex+1)))
	}

	if len(buttons) == 0 {
		return tgbotapi.NewInlineKeyboardMarkup()
	}
	return tgbotapi.NewInlineKeyboardMarkup(buttons)
}

func splitPages(text string, maxRunes int) []string {
	if maxRunes <= 0 {
		maxRunes = paginationMaxRunes
	}
	if utf8.RuneCountInString(text) <= maxRunes {
		return []string{text}
	}

	runes := []rune(text)
	pages := make([]string, 0)
	start := 0

	for start < len(runes) {
		remaining := len(runes) - start
		if remaining <= maxRunes {
			pages = append(pages, strings.TrimLeft(string(runes[start:]), "\n"))
			break
		}

		end := start + maxRunes
		cut := end
		for i := end; i > start+maxRunes/2; i-- {
			if runes[i-1] == '\n' {
				cut = i
				break
			}
		}

		chunk := strings.TrimLeft(string(runes[start:cut]), "\n")
		pages = append(pages, chunk)
		start = cut
	}

	return pages
}

package helpers

import (
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	"github.com/tudisco2005/telegram-torrent-bot/utils"
)

func AddTrackedIDs(h *handlers.Handler, ids []int) {
	if len(ids) == 0 {
		return
	}

	tracked, err := utils.LoadTracked(h.CompletedFilePath)
	if err != nil {
		h.Logger.Printf("[DEBUG] failed to load tracked ids file %s: %v", h.CompletedFilePath, err)
		return
	}

	seen := make(map[int]bool, len(tracked)+len(ids))
	for _, id := range tracked {
		seen[id] = true
	}

	changed := false
	for _, id := range ids {
		if !seen[id] {
			tracked = append(tracked, id)
			seen[id] = true
			changed = true
		}
	}

	if !changed {
		return
	}

	if err := utils.SaveTracked(h.CompletedFilePath, tracked); err != nil {
		h.Logger.Printf("[WARNING] failed to update %s: %v", h.CompletedFilePath, err)
	}
}

func RemoveTrackedIDs(h *handlers.Handler, ids []int) {
	if len(ids) == 0 {
		return
	}

	removeSet := make(map[int]bool, len(ids))
	for _, id := range ids {
		removeSet[id] = true
	}

	tracked, err := utils.LoadTracked(h.CompletedFilePath)
	if err != nil {
		h.Logger.Printf("[DEBUG] failed to load tracked ids file %s: %v", h.CompletedFilePath, err)
		return
	}

	newTracked := make([]int, 0, len(tracked))
	for _, id := range tracked {
		if !removeSet[id] {
			newTracked = append(newTracked, id)
		}
	}

	if len(newTracked) == len(tracked) {
		return
	}

	if err := utils.SaveTracked(h.CompletedFilePath, newTracked); err != nil {
		h.Logger.Printf("[WARNING] failed to update %s: %v", h.CompletedFilePath, err)
	}
}

func ClearTrackedIDs(h *handlers.Handler) {
	if err := utils.SaveTracked(h.CompletedFilePath, []int{}); err != nil {
		h.Logger.Printf("[WARNING] failed to clear %s: %v", h.CompletedFilePath, err)
	}
}

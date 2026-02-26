package commands

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	"github.com/tudisco2005/telegram-torrent-bot/utils"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Move copies completed downloads from the Transmission download directory
// (DefaultDownloadLocation / TRANSMISSION_DOWNLOAD_LOCATION) to DEFAULT_MOVE_LOCATION
func Move(h *handlers.Handler, ud tgbotapi.Update, tokens []string, cmd string) {

	src := h.DefaultDownloadLocation
	if src == "" {
		src = os.Getenv("TRANSMISSION_DOWNLOAD_LOCATION")
	}
	dst := h.DefaultMoveLocation
	if dst == "" {
		dst = os.Getenv("DEFAULT_MOVE_LOCATION")
	}

	if src == "" || dst == "" {
		h.SendWithFormat(ud.Message.Chat.ID, "move: source or destination not configured", cmd)
		return
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "move: failed to create destination: "+err.Error(), cmd)
		return
	}

	movedFile := filepath.Join(dst, "../moved.json")

	moved := make(map[string]map[string]string)
	if data, err := os.ReadFile(movedFile); err == nil {
		_ = json.Unmarshal(data, &moved)
	}

	if len(tokens) > 0 {
		tk := strings.ToLower(tokens[0])
		if tk == "?" {
			help := "move options:\n" +
				"- `move` : list move status for torrents\n" +
				"- `move all` : move all not-yet-moved downloads\n" +
				"- `move <id> [id2 ...]` : move data of specific torrent ids (or filenames)\n" +
				"- `move reset` : clear moved.json records\n" +
				"- `move clear` : delete all files/dirs in `DEFAULT_MOVE_LOCATION` (lists deleted files)"
			h.SendWithFormat(ud.Message.Chat.ID, help, cmd)
			return
		}
		if tk == "reset" {
			moved = make(map[string]map[string]string)
			if b, err := json.MarshalIndent(moved, "", "  "); err == nil {
				if werr := os.WriteFile(movedFile, b, 0644); werr != nil {
					h.SendWithFormat(ud.Message.Chat.ID, "move: failed to clear moved.json: "+werr.Error(), cmd)
				} else {
					h.SendWithFormat(ud.Message.Chat.ID, "move: moved.json cleared", cmd)
				}
			} else {
				h.SendWithFormat(ud.Message.Chat.ID, "move: failed to reset moved.json: "+err.Error(), cmd)
			}
			h.Logger.Printf("[DEBUG] Move: moved.json reset by user command")
			return
		}
		if tk == "clear" {

			dstEntries, derr := os.ReadDir(dst)
			if derr != nil {
				h.SendWithFormat(ud.Message.Chat.ID, "move: failed to read destination: "+derr.Error(), cmd)
				return
			}

			h.SendWithFormat(ud.Message.Chat.ID, "clearing destination...", cmd)

			var deleted []string
			var deleteErrs []string
			for _, ent := range dstEntries {
				name := ent.Name()
				if strings.HasPrefix(name, ".") || name == "moved.json" || strings.Contains(name, ".part") || strings.HasSuffix(name, ".crdownload") {
					continue
				}
				p := filepath.Join(dst, name)
				if err := os.RemoveAll(p); err != nil {
					deleteErrs = append(deleteErrs, fmt.Sprintf("%s: %v", name, err))
				} else {
					deleted = append(deleted, name)
				}
			}
			msg := "move: cleared destination"
			if len(deleted) > 0 {
				msg = msg + ": deleted:\n- " + strings.Join(utils.EscapeFileMDList(deleted), "\n- ")
			} else {
				msg = msg + ": nothing deleted"
			}
			if len(deleteErrs) > 0 {
				msg = msg + "\nErrors: " + strings.Join(deleteErrs, "; ")
			}

			moved = make(map[string]map[string]string)
			if b, err := json.MarshalIndent(moved, "", "  "); err == nil {
				if werr := os.WriteFile(movedFile, b, 0644); werr != nil {
					h.Logger.Printf("[WARNING] Move: failed to clear moved.json after clear command: %v", werr)
				} else {
					h.Logger.Printf("[DEBUG] Move: cleared moved.json after clear command")
				}
			} else {
				h.Logger.Printf("[WARNING] Move: failed to marshal empty moved.json after clear command: %v", err)
			}

			h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
			h.Logger.Printf("[DEBUG] Move: destination cleared by user command, deleted %d entries", len(deleted))
			return
		}
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*move:* failed to read source directory: "+err.Error(), cmd)
		return
	}

	if len(tokens) == 0 {

		h.SendWithFormat(ud.Message.Chat.ID, "Processing move: listing downloads...", cmd)
		torrents, terr := h.Client.GetTorrents()
		if terr != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*move:* failed to get torrents: "+terr.Error(), cmd)
			return
		}
		var lines []string
		for _, t := range torrents {
			name := t.Name
			status := "❌"

			if _, err := os.Stat(filepath.Join(src, name)); err == nil {
				if rec, ok := moved[name]; ok {
					if _, ok2 := rec["moved_at"]; ok2 {
						status = "✅"
					} else {
						status = "✅"
					}
				} else {
					status = "💾"
				}
			} else {

				status = "❓"
			}
			lines = append(lines, fmt.Sprintf("%s `<%d>` %s", status, t.ID, utils.EscapeFileMD(name)))
		}
		if len(lines) == 0 {
			h.SendWithFormat(ud.Message.Chat.ID, "move: no torrents found", cmd)
			return
		}
		cont := strings.Join(lines, "\n")
		fmt.Printf("%s\n", cont)
		h.SendWithFormat(ud.Message.Chat.ID, cont, cmd)
		return
	}

	dstHashes := make(map[string]string)
	dstEntries, derr := os.ReadDir(dst)
	if derr == nil {
		for _, dent := range dstEntries {
			dname := dent.Name()
			if strings.HasPrefix(dname, ".") || strings.Contains(dname, ".part") || strings.HasSuffix(dname, ".crdownload") || dname == "moved.json" {
				continue
			}
			dpath := filepath.Join(dst, dname)
			if hsh, err := computePathHash(dpath); err == nil {
				dstHashes[hsh] = dname
			} else {
				h.Logger.Printf("[DEBUG] Move: failed to hash destination %s: %v", dpath, err)
			}
		}
	}
	h.Logger.Printf("[DEBUG] Move: computed destination hashes for %d entries", len(dstHashes))

	var toProcess []string // names to process

	if len(tokens) > 0 && strings.ToLower(tokens[0]) == "all" {
		for _, ent := range entries {
			name := ent.Name()
			if strings.HasPrefix(name, ".") || strings.Contains(name, ".part") || strings.HasSuffix(name, ".crdownload") {
				continue
			}
			if _, ok := moved[name]; ok {
				continue
			}
			toProcess = append(toProcess, name)
		}
		h.Logger.Printf("[DEBUG] Move: token 'all' detected, %d entries to process", len(toProcess))
	} else if len(tokens) > 0 {

		for _, tk := range tokens {
			id, err := strconv.Atoi(tk)
			if err != nil {

				toProcess = append(toProcess, tk)
				continue
			}
			t, terr := h.Client.GetTorrent(id)
			if terr != nil {
				h.SendWithFormat(ud.Message.Chat.ID, fmt.Sprintf("*move:* failed to lookup torrent id %d: %v", id, terr), cmd)
				continue
			}

			found := false
			for _, ent := range entries {
				if ent.Name() == t.Name {
					toProcess = append(toProcess, ent.Name())
					found = true
					break
				}
			}
			if !found {
				h.SendWithFormat(ud.Message.Chat.ID, fmt.Sprintf("*move:* source entry for torrent id %d (%s) not found", id, t.Name), cmd)
			}
		}
	} else {

		for _, ent := range entries {
			name := ent.Name()
			if strings.HasPrefix(name, ".") || strings.Contains(name, ".part") || strings.HasSuffix(name, ".crdownload") {
				continue
			}
			if _, ok := moved[name]; ok {
				continue
			}
			toProcess = append(toProcess, name)
		}
	}

	var copied []string
	var errorsList []string
	skippedDuplicates := 0
	failedCount := 0

	if len(toProcess) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "move: no completed downloads found to copy", cmd)
		return
	}

	progressStartedAt := time.Now()
	progressMsgID := h.SendWithFormat(ud.Message.Chat.ID, buildMoveProgressMessage(0, len(toProcess), 0, 0, 0, progressStartedAt), cmd, "markdown")
	lastProgressUpdate := time.Now()

	for i, name := range toProcess {
		sPath := filepath.Join(src, name)
		dPath := filepath.Join(dst, name)

		sHash, err := computePathHash(sPath)
		if err != nil {
			errorsList = append(errorsList, fmt.Sprintf("%s: failed to compute hash: %v", name, err))
			failedCount++
			if progressMsgID > 0 {
				done := i + 1
				if done == len(toProcess) || time.Since(lastProgressUpdate) >= 700*time.Millisecond {
					updateMoveProgressMessage(h, ud.Message.Chat.ID, progressMsgID, buildMoveProgressMessage(done, len(toProcess), len(copied), skippedDuplicates, failedCount, progressStartedAt))
					lastProgressUpdate = time.Now()
				}
			}
			continue
		}
		if existing, ok := dstHashes[sHash]; ok {
			h.Logger.Printf("[INFO] Move: skipping %s - duplicate of destination %s (hash)", sPath, existing)

			errorsList = append(errorsList, fmt.Sprintf("%s: duplicate of %s (skipped)", name, existing))
			skippedDuplicates++

			moved[name] = map[string]string{"moved_at": time.Now().Format(time.RFC3339), "dest": existing, "hash": sHash}
			if progressMsgID > 0 {
				done := i + 1
				if done == len(toProcess) || time.Since(lastProgressUpdate) >= 700*time.Millisecond {
					updateMoveProgressMessage(h, ud.Message.Chat.ID, progressMsgID, buildMoveProgressMessage(done, len(toProcess), len(copied), skippedDuplicates, failedCount, progressStartedAt))
					lastProgressUpdate = time.Now()
				}
			}
			continue
		}

		if err := copyPath(sPath, dPath); err != nil {
			errorsList = append(errorsList, fmt.Sprintf("%s: %v", name, err))
			failedCount++
			if progressMsgID > 0 {
				done := i + 1
				if done == len(toProcess) || time.Since(lastProgressUpdate) >= 700*time.Millisecond {
					updateMoveProgressMessage(h, ud.Message.Chat.ID, progressMsgID, buildMoveProgressMessage(done, len(toProcess), len(copied), skippedDuplicates, failedCount, progressStartedAt))
					lastProgressUpdate = time.Now()
				}
			}
			continue
		}

		moved[name] = map[string]string{"moved_at": time.Now().Format(time.RFC3339), "dest": dPath, "hash": sHash}
		copied = append(copied, name)

		if progressMsgID > 0 {
			done := i + 1
			if done == len(toProcess) || time.Since(lastProgressUpdate) >= 700*time.Millisecond {
				updateMoveProgressMessage(h, ud.Message.Chat.ID, progressMsgID, buildMoveProgressMessage(done, len(toProcess), len(copied), skippedDuplicates, failedCount, progressStartedAt))
				lastProgressUpdate = time.Now()
			}
		}
	}

	if progressMsgID > 0 {
		updateMoveProgressMessage(h, ud.Message.Chat.ID, progressMsgID, buildMoveProgressMessage(len(toProcess), len(toProcess), len(copied), skippedDuplicates, failedCount, progressStartedAt)+"\nmove completed.")
	}

	if b, err := json.MarshalIndent(moved, "", "  "); err == nil {
		_ = os.WriteFile(movedFile, b, 0644)
	} else {
		h.Logger.Printf("[WARNING] Move: failed to save moved.json: %v", err)
	}

	if len(copied) == 0 {
		if len(errorsList) > 0 {
			h.SendWithFormat(ud.Message.Chat.ID, "move: errors: "+strings.Join(errorsList, "; "), cmd)
			return
		}
		h.SendWithFormat(ud.Message.Chat.ID, "move: no completed downloads found to copy", cmd)
		return
	}

	msg := fmt.Sprintf("move: copied %d item(s) to %s\n- %s", len(copied), utils.EscapeFileMD(dst), strings.Join(utils.EscapeFileMDList(copied), "\n- "))
	if len(errorsList) > 0 {
		msg = msg + "\nErrors: " + strings.Join(errorsList, "; ")
	}
	h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
}

func updateMoveProgressMessage(h *handlers.Handler, chatID int64, messageID int, text string) {
	if messageID <= 0 {
		return
	}
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = tgbotapi.ModeMarkdown
	if _, err := h.Bot.Send(edit); err != nil {
		h.Logger.Printf("[DEBUG] Move: failed to update progress message: chat=%d msg=%d err=%v", chatID, messageID, err)
	}
}

func buildMoveProgressMessage(done, total, copied, skipped, failed int, startedAt time.Time) string {
	return fmt.Sprintf("move started...\n%s\n%d/%d processed | copied: %d | skipped: %d | failed: %d", buildMoveProgressBar(done, total, startedAt), done, total, copied, skipped, failed)
}

func buildMoveProgressBar(done, total int, startedAt time.Time) string {
	const width = 15
	if total <= 0 {
		return fmt.Sprintf("`%s` 0%% ETA: n/a", utils.ProgressBar(0, width))
	}
	if done < 0 {
		done = 0
	}
	if done > total {
		done = total
	}
	bar := utils.ProgressBar(float64(done)/float64(total), width)
	percent := (done * 100) / total

	eta := "n/a"
	if done > 0 && done < total && !startedAt.IsZero() {
		elapsed := time.Since(startedAt)
		if elapsed < 0 {
			elapsed = 0
		}
		elapsedPerItem := elapsed / time.Duration(done)
		remaining := total - done
		etaDuration := elapsedPerItem * time.Duration(remaining)
		eta = formatETA(etaDuration)
	}

	return fmt.Sprintf("`%s` %d%% ETA: %s", bar, percent, eta)
}

func formatETA(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	seconds := int(d.Round(time.Second).Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	secs := seconds % 60
	if minutes < 60 {
		return fmt.Sprintf("%dm %ds", minutes, secs)
	}
	hours := minutes / 60
	mins := minutes % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}

func copyPath(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func computePathHash(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	h := sha1.New()
	if info.IsDir() {
		// For directories, hash the directory name and its contents' names
		h.Write([]byte(info.Name()))
		entries, err := os.ReadDir(path)
		if err != nil {
			return "", err
		}
		for _, entry := range entries {
			h.Write([]byte(entry.Name()))
		}
	} else {
		// For files, hash the file name and size
		h.Write([]byte(info.Name()))
		h.Write([]byte(fmt.Sprintf("%d", info.Size())))
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

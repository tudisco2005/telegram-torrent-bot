package handlers

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/tudisco2005/telegram-torrent-bot/utils"
	"github.com/dustin/go-humanize"
	"github.com/pyed/transmission"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Add adds torrents from URLs or magnets via Transmission
func (h *Handler) Add(ud tgbotapi.Update, tokens []string, cmd string) {
	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*add:* needs at least one URL or magnet", cmd)
		return
	}

	var buf bytes.Buffer
	// Get existing torrents once for comparison
	existingTorrents, _ := h.Client.GetTorrents()
	existingIDs := make(map[int]bool)
	existingCompleted := make(map[int]bool)
	for i := range existingTorrents {
		existingIDs[existingTorrents[i].ID] = true
		// Mark as completed if PercentDone is 1.0 (100%)
		if existingTorrents[i].PercentDone >= 1.0 {
			existingCompleted[existingTorrents[i].ID] = true
		}
	}

	for _, link := range tokens {
		// Validate that input is a magnet link or HTTP(S) URL
		if !strings.HasPrefix(link, "magnet") && !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
			buf.WriteString("*add:* invalid URL or magnet link — " + link + "\n")
			continue
		}

		// proceed to add the link (no duplicate-name check)

		addCmd := transmission.NewAddCmdByURL(link)
		h.Logger.Printf("[DEBUG] Add: attempting to add link=%s", link)
		added, err := h.Client.ExecuteAddCommand(addCmd)
		if err != nil {
			buf.WriteString("*add:* " + err.Error() + " — " + link + "\n")
			continue
		}

		// Debug: log the returned added struct and URL to help diagnose multiple empty names
		h.Logger.Printf("[DEBUG] Add: url=%s added=%#v existingID=%v", link, added, existingIDs[added.ID])

		// no duplicate-ID skip: always report added result

		// If available space is less than the torrent size, stop it and notify
		path := h.DefaultDownloadLocation
		if path == "" {
			path = "/"
		}
		var stat syscall.Statfs_t
		if err := syscall.Statfs(path, &stat); err == nil {
			avail := uint64(stat.Bavail) * uint64(stat.Bsize)

			// Query torrent info to get size when done
			if torrentFull, err := h.Client.GetTorrent(added.ID); err == nil {
				// If size unknown (magnet without metadata), allow Transmission to fetch metadata.
				// Start a background poller to act once metadata becomes available.
				if torrentFull.SizeWhenDone == 0 {
					h.SendWithFormat(ud.Message.Chat.ID, "Added magnet: metadata not available yet — letting Transmission fetch metadata", cmd)
					go func(tid int, link string) {
						for i := 0; i < 60; i++ { // ~5 minutes (60 * 5s)
							time.Sleep(5 * time.Second)
							t, err := h.Client.GetTorrent(tid)
							if err != nil {
								break
							}
							// If transmission reports an error during metadata fetch, stop polling
							if t.Error != 0 || strings.Contains(strings.ToLower(t.ErrorString), "no space") {
								// attempt to delete to avoid cruft
								if _, derr := h.Client.DeleteTorrent(tid, false); derr != nil {
									h.Logger.Printf("[WARNING] failed to delete torrent id=%d after error: %v", tid, derr)
								}
								h.SendWithFormat(ud.Message.Chat.ID, "*add:* "+t.ErrorString+" — "+link, cmd)
								return
							}
							if t.SizeWhenDone > 0 {
								var need uint64
								if t.Have() > 0 {
									need = uint64(t.SizeWhenDone) - uint64(t.Have())
								} else {
									need = uint64(t.SizeWhenDone)
								}
								var st syscall.Statfs_t
								if err := syscall.Statfs(path, &st); err == nil {
									availNow := uint64(st.Bavail) * uint64(st.Bsize)
									if need > 0 && availNow < need {
										// Not enough space: stop the torrent to avoid filling disk
										if _, serr := h.Client.StopTorrent(tid); serr == nil {
											h.SendWithFormat(ud.Message.Chat.ID, fmt.Sprintf("Not enough space for torrent %s (id=%d); torrent stopped", t.Name, tid), cmd)
										}
										return
									}
								}
								// Metadata fetched and there is space (or we couldn't statfs): notify user
								h.SendWithFormat(ud.Message.Chat.ID, fmt.Sprintf("Metadata fetched for torrent %s (id=%d)", t.Name, tid), cmd)
								return
							}
						}
						// metadata never arrived in time
						h.SendWithFormat(ud.Message.Chat.ID, "Metadata not available after timeout — no action taken", cmd)
					}(added.ID, link)
					continue
				}
				if torrentFull.SizeWhenDone > 0 && avail < uint64(torrentFull.SizeWhenDone) {
					// pause/stop the torrent and inform the user
					h.Client.StopTorrent(added.ID)
					h.SendWithFormat(ud.Message.Chat.ID, "Not enough space left, torrent stopped", cmd)
					continue
				}
				// If Transmission reported an error (e.g. unable to write resume file)
				// remove the torrent and report the error back to the user.
				if torrentFull.Error != 0 || strings.Contains(strings.ToLower(torrentFull.ErrorString), "no space") {
					// Attempt to delete the torrent record to clean up
					if _, derr := h.Client.DeleteTorrent(added.ID, false); derr != nil {
						h.Logger.Printf("[WARNING] failed to delete torrent id=%d after error: %v", added.ID, derr)
					}
					errMsg := torrentFull.ErrorString
					if errMsg == "" {
						errMsg = "Not enough space left (transmission reported error)"
					}
					h.SendWithFormat(ud.Message.Chat.ID, "*add:* "+errMsg+" — "+link, cmd)
					continue
				}
			}
		}

		// collect successful adds and send a single combined message later
		buf.WriteString(h.FormatOutputString(cmd, added.Name))
		// ensure newline separation when FormatOutputString does not include it
		if !strings.HasSuffix(buf.String(), "\n") {
			buf.WriteString("\n")
		}
	}

	if buf.Len() > 0 {
		// Trim trailing newline
		out := strings.TrimRight(buf.String(), "\n")
		h.SendWithFormat(ud.Message.Chat.ID, out, cmd)
	}
}

// ReceiveTorrent handles torrent file uploads: saves to DEFAULT_TORRENT_LOCATION then adds to Transmission
func (h *Handler) ReceiveTorrent(ud tgbotapi.Update) {
	if ud.Message.Document == nil {
		return
	}
	h.Logger.Printf("[DEBUG] ReceiveTorrent: document received FileID=%s FileName=%q Size=%d",
		ud.Message.Document.FileID, ud.Message.Document.FileName, ud.Message.Document.FileSize)

	if h.DefaultTorrentLocation == "" {
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* DEFAULT_TORRENT_LOCATION is not set", "add")
		return
	}

	fconfig := tgbotapi.FileConfig{FileID: ud.Message.Document.FileID}
	file, err := h.Bot.GetFile(fconfig)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* "+err.Error(), "add")
		return
	}
	h.Logger.Printf("[DEBUG] ReceiveTorrent: GetFile ok FilePath=%s", file.FilePath)

	// Download file from Telegram
	downloadURL := file.Link(h.BotToken)
	h.Logger.Printf("[DEBUG] ReceiveTorrent: downloading from Telegram (FilePath=%s)", file.FilePath)
	resp, err := http.Get(downloadURL)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* download failed: "+err.Error(), "add")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* download failed: status "+resp.Status, "add")
		return
	}
	h.Logger.Printf("[DEBUG] ReceiveTorrent: download ok status=%d content_length=%d", resp.StatusCode, resp.ContentLength)

	// Ensure save directory exists
	h.Logger.Printf("[DEBUG] ReceiveTorrent: ensuring directory exists: %s", h.DefaultTorrentLocation)
	if err := os.MkdirAll(h.DefaultTorrentLocation, 0755); err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* "+err.Error(), "add")
		return
	}

	// Save to DEFAULT_TORRENT_LOCATION (use document filename or fallback)
	name := ud.Message.Document.FileName
	if name == "" {
		name = filepath.Base(file.FilePath)
	}
	if name == "" || name == "." {
		name = "torrent_" + ud.Message.Document.FileID + ".torrent"
	}
	if !strings.HasSuffix(strings.ToLower(name), ".torrent") {
		name = name + ".torrent"
	}
	savePath := filepath.Join(h.DefaultTorrentLocation, name)
	h.Logger.Printf("[DEBUG] ReceiveTorrent: saving .torrent as %s", savePath)

	out, err := os.Create(savePath)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* "+err.Error(), "add")
		return
	}
	n, err := io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(savePath)
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* "+err.Error(), "add")
		return
	}
	h.Logger.Printf("[DEBUG] ReceiveTorrent: saved %d bytes to %s", n, savePath)

	// Add to Transmission from the saved file
	h.Logger.Printf("[DEBUG] ReceiveTorrent: adding to Transmission from file %s", savePath)
	// proceed to add uploaded .torrent file (no duplicate-name check)

	addCmd, err := transmission.NewAddCmdByFile(savePath)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* "+err.Error(), "add")
		return
	}
	added, err := h.Client.ExecuteAddCommand(addCmd)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* "+err.Error(), "add")
		return
	}
	h.Logger.Printf("[DEBUG] ReceiveTorrent: added to Transmission id=%d name=%q", added.ID, added.Name)

	// Check available space and stop torrent if insufficient
	path := h.DefaultDownloadLocation
	if path == "" {
		path = "/"
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err == nil {
		avail := uint64(stat.Bavail) * uint64(stat.Bsize)
		if torrentFull, err := h.Client.GetTorrent(added.ID); err == nil {
			if torrentFull.SizeWhenDone == 0 {
				h.SendWithFormat(ud.Message.Chat.ID, "Added torrent: metadata not available yet — letting Transmission fetch metadata", "add")
				// Poll for metadata and decide later
				go func(tid int) {
					for i := 0; i < 60; i++ {
						time.Sleep(5 * time.Second)
						t, err := h.Client.GetTorrent(tid)
						if err != nil {
							break
						}
						if t.Error != 0 || strings.Contains(strings.ToLower(t.ErrorString), "no space") {
							if _, derr := h.Client.DeleteTorrent(tid, false); derr != nil {
								h.Logger.Printf("[WARNING] failed to delete torrent id=%d after error: %v", tid, derr)
							}
							h.SendWithFormat(ud.Message.Chat.ID, "*add:* "+t.ErrorString, "add")
							return
						}
						if t.SizeWhenDone > 0 {
							var need uint64
							if t.Have() > 0 {
								need = uint64(t.SizeWhenDone) - uint64(t.Have())
							} else {
								need = uint64(t.SizeWhenDone)
							}
							var st syscall.Statfs_t
							if err := syscall.Statfs(path, &st); err == nil {
								availNow := uint64(st.Bavail) * uint64(st.Bsize)
								if need > 0 && availNow < need {
									// Not enough space: stop the torrent to avoid filling disk
									if _, serr := h.Client.StopTorrent(tid); serr == nil {
										h.SendWithFormat(ud.Message.Chat.ID, fmt.Sprintf("Not enough space for torrent %s (id=%d); torrent stopped", t.Name, tid), "add")
									}
									return
								}
							}
							// Metadata fetched and there is space (or we couldn't statfs): notify user
							h.SendWithFormat(ud.Message.Chat.ID, fmt.Sprintf("Metadata fetched for torrent %s (id=%d)", t.Name, tid), "add")
							return
						}
					}
					h.SendWithFormat(ud.Message.Chat.ID, "Metadata not available after timeout — no action taken", "add")
				}(added.ID)
			}
			if torrentFull.SizeWhenDone > 0 && avail < uint64(torrentFull.SizeWhenDone) {
				h.Client.StopTorrent(added.ID)
				h.SendWithFormat(ud.Message.Chat.ID, "Not enough space left, torrent stopped", "add")
				return
			}
			if torrentFull.Error != 0 || strings.Contains(strings.ToLower(torrentFull.ErrorString), "no space") {
				if _, derr := h.Client.DeleteTorrent(added.ID, false); derr != nil {
					h.Logger.Printf("[WARNING] failed to delete torrent id=%d after error: %v", added.ID, derr)
				}
				errMsg := torrentFull.ErrorString
				if errMsg == "" {
					errMsg = "Not enough space left (transmission reported error)"
				}
				h.SendWithFormat(ud.Message.Chat.ID, "*add:* "+errMsg, "add")
				return
			}
		}
	}

	msg := h.FormatOutputString("add", added.Name)
	h.SendWithFormat(ud.Message.Chat.ID, msg, "add")
}

// Search searches for torrents by name
func (h *Handler) Search(ud tgbotapi.Update, tokens []string, cmd string) {
	// make sure that we got a query
	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*search:* needs an argument", cmd)
		return
	}

	query := strings.Join(tokens, " ")
	// "(?i)" for case insensitivity
	regx, err := regexp.Compile("(?i)" + query)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*search:* "+err.Error(), cmd)
		return
	}

	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*search:* "+err.Error(), cmd)
		return
	}

	buf := new(bytes.Buffer)
	for i := range torrents {
		if regx.MatchString(torrents[i].Name) {
			buf.WriteString(h.FormatOutputString(cmd, torrents[i].ID, torrents[i].Name))
		}
	}
	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "No matches!", cmd)
		return
	}
	h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)
}

// Latest lists the latest n added torrents
func (h *Handler) Latest(ud tgbotapi.Update, tokens []string, cmd string) {
	var (
		n   = 5 // default to 5
		err error
	)

	if len(tokens) > 0 {
		n, err = strconv.Atoi(tokens[0])
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*latest:* "+err.Error(), cmd)
			return
		}
	}

	torrents, err := h.Client.GetTorrents()
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*latest:* "+err.Error(), cmd)
		return
	}

	// make sure that we stay in the boundaries
	if n <= 0 || n > len(torrents) {
		n = len(torrents)
	}

	// sort by age, and set reverse to true to get the latest first
	torrents.SortAge(true)

	buf := new(bytes.Buffer)
	for i := range torrents[:n] {
		buf.WriteString(h.FormatOutputString(cmd, torrents[i].ID, torrents[i].Name))
	}
	if buf.Len() == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*latest:* No torrents", cmd, "markdown")
		return
	}
	h.SendWithFormat(ud.Message.Chat.ID, buf.String(), cmd)
}

// Info shows detailed information about a torrent
func (h *Handler) Info(ud tgbotapi.Update, tokens []string, cmd string) {
	if len(tokens) == 0 {
		h.SendWithFormat(ud.Message.Chat.ID, "*info:* needs a torrent ID number", cmd)
		return
	}

	for _, id := range tokens {
		torrentID, err := strconv.Atoi(id)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*info:* "+err.Error(), cmd)
			continue
		}

		// get the torrent
		torrent, err := h.Client.GetTorrent(torrentID)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*info:* "+err.Error(), cmd)
			continue
		}

		// get the trackers using 'trackerRegex'
		var trackers string
		for _, tracker := range torrent.Trackers {
			if matches := trackerRegex.FindStringSubmatch(tracker.Announce); matches != nil {
				trackers += matches[1] + ", "
			}
		}

		// format the info
		torrentName := h.Replacer.Replace(torrent.Name) // escape markdown
		info := fmt.Sprintf("`<%d>` *%s*\n%s *%s* of *%s* (*%.1f%%*) ↓ *%s*  ↑ *%s* R: *%s*\nDL: *%s* UP: *%s*\nAdded: *%s*, ETA: *%s*\nTrackers: `%s`",
			torrent.ID, torrentName, torrent.TorrentStatus(), humanize.Bytes(torrent.Have()), humanize.Bytes(torrent.SizeWhenDone),
			torrent.PercentDone*100, humanize.Bytes(torrent.RateDownload), humanize.Bytes(torrent.RateUpload), torrent.Ratio(),
			humanize.Bytes(torrent.DownloadedEver), humanize.Bytes(torrent.UploadedEver), time.Unix(torrent.AddedDate, 0).Format(time.Stamp),
			torrent.ETA(), trackers)

		// send it
		msgID := h.SendWithFormat(ud.Message.Chat.ID, info, cmd)

		if h.NoLive || h.UpdateMaxIterations == 0 {
			continue
		}

		iterations := h.Duration
		if h.UpdateMaxIterations > 0 && h.UpdateMaxIterations < iterations {
			iterations = h.UpdateMaxIterations
		}
		chatID := ud.Message.Chat.ID
		// this go-routine will make the info live for 'iterations * interval'
		go func(torrentID, msgID int, chatID int64) {
			for i := 0; i < iterations; i++ {
				time.Sleep(h.Interval)

				torrent, err := h.Client.GetTorrent(torrentID)
				if err != nil {
					break
				}

				torrentName := h.Replacer.Replace(torrent.Name) // escape markdown
				info := fmt.Sprintf("`<%d>` *%s*\n%s *%s* of *%s* (*%.1f%%*) ↓ *%s*  ↑ *%s* R: *%s*\nDL: *%s* UP: *%s*\nAdded: *%s*, ETA: *%s*\nTrackers: `%s`",
					torrent.ID, torrentName, torrent.TorrentStatus(), humanize.Bytes(torrent.Have()), humanize.Bytes(torrent.SizeWhenDone),
					torrent.PercentDone*100, humanize.Bytes(torrent.RateDownload), humanize.Bytes(torrent.RateUpload), torrent.Ratio(),
					humanize.Bytes(torrent.DownloadedEver), humanize.Bytes(torrent.UploadedEver), time.Unix(torrent.AddedDate, 0).Format(time.Stamp),
					torrent.ETA(), trackers)

				editConf := tgbotapi.NewEditMessageText(chatID, msgID, info)
				editConf.ParseMode = tgbotapi.ModeMarkdown
				if resp, err := h.Bot.Send(editConf); err != nil {
					h.Logger.Printf("[DEBUG] EditMessage failed: ChatID=%d MsgID=%d Len=%d Err=%v", chatID, msgID, len(info), err)
				} else {
					h.Logger.Printf("[DEBUG] EditMessage sent: ChatID=%d MsgID=%d RespMessageID=%d", chatID, msgID, resp.MessageID)
				}
			}
			// sleep one more time before the dashes
			time.Sleep(h.Interval)

			// at the end write dashes to indicate that we are done being live.
			torrent, err := h.Client.GetTorrent(torrentID)
			if err != nil {
				return
			}
			torrentName := h.Replacer.Replace(torrent.Name) // escape markdown
			info := fmt.Sprintf("`<%d>` *%s*\n%s *%s* of *%s* (*-%%*) ↓ *-*  ↑ *-* R: *-*\nDL: *-* UP: *-*\nAdded: *%s*, ETA: *-*\nTrackers: `%s`",
				torrent.ID, torrentName, torrent.TorrentStatus(), humanize.Bytes(torrent.Have()), humanize.Bytes(torrent.SizeWhenDone),
				time.Unix(torrent.AddedDate, 0).Format(time.Stamp), trackers)

			editConf := tgbotapi.NewEditMessageText(chatID, msgID, info)
			editConf.ParseMode = tgbotapi.ModeMarkdown
			if resp, err := h.Bot.Send(editConf); err != nil {
				h.Logger.Printf("[DEBUG] EditMessage failed: ChatID=%d MsgID=%d Len=%d Err=%v", chatID, msgID, len(info), err)
			} else {
				h.Logger.Printf("[DEBUG] EditMessage sent: ChatID=%d MsgID=%d RespMessageID=%d", chatID, msgID, resp.MessageID)
			}
		}(torrentID, msgID, chatID)
	}
}

// Move copies completed downloads from the Transmission download directory
// (DefaultDownloadLocation / TRANSMISSION_DONWNLOAD_LOCATION) to DEFAULT_MOVE_LOCATION
func (h *Handler) Move(ud tgbotapi.Update, tokens []string, cmd string) {
	// determine source and destination directories
	src := h.DefaultDownloadLocation
	if src == "" {
		src = os.Getenv("TRANSMISSION_DONWNLOAD_LOCATION")
	}
	dst := h.DefaultMoveLocation
	if dst == "" {
		dst = os.Getenv("DEFAULT_MOVE_LOCATION")
	}

	if src == "" || dst == "" {
		h.SendWithFormat(ud.Message.Chat.ID, "move: source or destination not configured", cmd)
		return
	}

	// ensure destination exists
	if err := os.MkdirAll(dst, 0755); err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "move: failed to create destination: "+err.Error(), cmd)
		return
	}

	// path to moved.json inside destination
	movedFile := filepath.Join(dst, "../moved.json")

	// load moved records (map[name] -> map[string]string)
	moved := make(map[string]map[string]string)
	if data, err := os.ReadFile(movedFile); err == nil {
		_ = json.Unmarshal(data, &moved)
	}

	// support clearing/resetting the moved.json via `move reset` or `move clear`
	if len(tokens) > 0 {
		tk := strings.ToLower(tokens[0])
		if tk == "reset" || tk == "clear" {
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
			h.Logger.Printf("[DEBUG] Move: moved.json reset/cleared by user command")
			return
		}
	}

	// list entries in source
	entries, err := os.ReadDir(src)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*move:* failed to read source directory: "+err.Error(), cmd)
		return
	}

	// If no token provided, list all torrents and their move status
	if len(tokens) == 0 {
		// quick ack so user sees a response immediately
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
			// check if source entry exists
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
				// no source file; mark as error/absent
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

	// Build fingerprint map for destination entries to detect duplicates by content hash
	dstHashes := make(map[string]string) // hash -> dst entry name
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
	// If token is "all", move all not-yet-moved entries
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
		// treat tokens as torrent IDs; map id -> torrent name and add to toProcess if exists in source
		for _, tk := range tokens {
			id, err := strconv.Atoi(tk)
			if err != nil {
				// not a number, maybe a direct filename
				toProcess = append(toProcess, tk)
				continue
			}
			t, terr := h.Client.GetTorrent(id)
			if terr != nil {
				h.SendWithFormat(ud.Message.Chat.ID, fmt.Sprintf("*move:* failed to lookup torrent id %d: %v", id, terr), cmd)
				continue
			}
			// find matching entry in source by name
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
		// default: same as "all"
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

	for _, name := range toProcess {
		sPath := filepath.Join(src, name)
		dPath := filepath.Join(dst, name)

		// compute hash of source entry and check for a matching hash in destination
		sHash, err := computePathHash(sPath)
		if err != nil {
			errorsList = append(errorsList, fmt.Sprintf("%s: failed to compute hash: %v", name, err))
			continue
		}
		if existing, ok := dstHashes[sHash]; ok {
			h.Logger.Printf("[INFO] Move: skipping %s - duplicate of destination %s (hash)", sPath, existing)
			// record as skipped
			errorsList = append(errorsList, fmt.Sprintf("%s: duplicate of %s (skipped)", name, existing))
			// mark as moved in moved.json to avoid future attempts
			moved[name] = map[string]string{"moved_at": time.Now().Format(time.RFC3339), "dest": existing, "hash": sHash}
			continue
		}

		if err := copyPath(sPath, dPath); err != nil {
			errorsList = append(errorsList, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		// record as moved
		moved[name] = map[string]string{"moved_at": time.Now().Format(time.RFC3339), "dest": dPath, "hash": sHash}
		copied = append(copied, name)
	}

	// save moved.json
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

	msg := fmt.Sprintf("move: copied %d item(s) to %s\n- %s", len(copied), dst, strings.Join(copied, "\n- "))
	if len(errorsList) > 0 {
		msg = msg + "\nErrors: " + strings.Join(errorsList, "; ")
	}
	h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
}

// copyPath copies a file or directory recursively from src to dst
func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(src, path)
			if err != nil {
				return err
			}
			target := filepath.Join(dst, rel)
			if fi.IsDir() {
				return os.MkdirAll(target, fi.Mode())
			}
			// copy file
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			in, err := os.Open(path)
			if err != nil {
				return err
			}
			defer in.Close()
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fi.Mode())
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, in); err != nil {
				out.Close()
				return err
			}
			return out.Close()
		})
	}
	// single file
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// computePathHash computes a deterministic SHA1 fingerprint for a file or directory.
// For files, it hashes the filename, size and contents. For directories it walks
// files in sorted order and hashes each file's relative path, size and contents.
func computePathHash(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	h := sha1.New()

	if !info.IsDir() {
		// single file: include base name and size
		rel := filepath.Base(path)
		if _, err := h.Write([]byte(rel)); err != nil {
			return "", err
		}
		if _, err := h.Write([]byte("\n")); err != nil {
			return "", err
		}
		if _, err := h.Write([]byte(fmt.Sprintf("%d\n", info.Size()))); err != nil {
			return "", err
		}
		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer f.Close()
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}
		return hex.EncodeToString(h.Sum(nil)), nil
	}

	// directory: collect files (not directories) in deterministic order
	var files []string
	err = filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(path, p)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(files)

	for _, rel := range files {
		p := filepath.Join(path, rel)
		fi, err := os.Stat(p)
		if err != nil {
			return "", err
		}
		if _, err := h.Write([]byte(rel)); err != nil {
			return "", err
		}
		if _, err := h.Write([]byte("\n")); err != nil {
			return "", err
		}
		if _, err := h.Write([]byte(fmt.Sprintf("%d\n", fi.Size()))); err != nil {
			return "", err
		}
		f, err := os.Open(p)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(h, f); err != nil {
			f.Close()
			return "", err
		}
		f.Close()
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}


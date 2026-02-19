package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/pyed/transmission"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// DownloadDir sets the download directory
func (h *Handler) DownloadDir(ud tgbotapi.Update, tokens []string, cmd string) {
	if len(tokens) < 1 {
		h.SendWithFormat(ud.Message.Chat.ID, "Please, specify a path for downloaddir", cmd)
		return
	}

	downloadDir := tokens[0]

	// Update in-memory download directory (session-wide). Persisting or applying
	// to Transmission per-torrent options is not implemented here.
	h.DefaultDownloadLocation = downloadDir
	msg := h.FormatOutputString(cmd, downloadDir)
	h.SendWithFormat(ud.Message.Chat.ID, msg, cmd)
}

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

		// best-effort duplicate detection: compare display name (dn) for magnets
		// or filename for http(s) .torrent URLs against existing torrent names
		var candidate string
		if strings.HasPrefix(link, "magnet") {
			// Extract display name (dn) from magnet link
			// Magnet format: magnet:?xt=...&dn=...&tr=...
			if idx := strings.Index(link, "?"); idx != -1 {
				queryStr := link[idx+1:]
				values, err := neturl.ParseQuery(queryStr)
				if err == nil {
					dn := values.Get("dn")
					if dn != "" {
						// URL decode the display name
						if decoded, err := neturl.QueryUnescape(dn); err == nil {
							candidate = decoded
						} else {
							candidate = dn
						}
					} else {
						// Fallback to xt (info hash) if dn is not available
						candidate = values.Get("xt")
					}
				}
			}
		} else if strings.HasPrefix(link, "http") {
			if u, err := neturl.Parse(link); err == nil {
				candidate = strings.TrimSuffix(filepath.Base(u.Path), ".torrent")
			}
		}

		// Check if candidate matches any existing torrent name (by name only, check all states)
		foundDuplicate := false
		if candidate != "" {
			for i := range existingTorrents {
				if strings.EqualFold(existingTorrents[i].Name, candidate) || strings.Contains(strings.ToLower(existingTorrents[i].Name), strings.ToLower(candidate)) {
					// Check if duplicate is already completed
					if existingCompleted[existingTorrents[i].ID] {
						buf.WriteString("duplicated detected, skipping\n")
					} else {
						buf.WriteString("duplicated detected (not completed), skipping\n")
					}
					foundDuplicate = true
					break
				}
			}
		}

		// If we found a duplicate by name, skip adding
		if foundDuplicate {
			continue
		}

		addCmd := transmission.NewAddCmdByURL(link)
		h.Logger.Printf("[DEBUG] Add: attempting to add link=%s candidate=%s", link, candidate)
		added, err := h.Client.ExecuteAddCommand(addCmd)
		if err != nil {
			buf.WriteString("*add:* " + err.Error() + " — " + link + "\n")
			continue
		}

		// Debug: log the returned added struct and URL to help diagnose multiple empty names
		h.Logger.Printf("[DEBUG] Add: url=%s added=%#v existingID=%v", link, added, existingIDs[added.ID])

		// Check if transmission returned an existing torrent ID (duplicate)
		if existingIDs[added.ID] {
			h.Logger.Printf("[DEBUG] Add: detected duplicate by ID=%d", added.ID)
			// Check if duplicate is already completed
			if existingCompleted[added.ID] {
				buf.WriteString("duplicated detected, skipping\n")
			} else {
				buf.WriteString("duplicated detected (not completed), skipping\n")
			}
			continue
		}

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
	// Check by filename against existing torrents to avoid duplicates
	candidate := strings.TrimSuffix(name, ".torrent")
	if candidate != "" {
		if torrents, err := h.Client.GetTorrents(); err == nil {
			for i := range torrents {
				if strings.EqualFold(torrents[i].Name, candidate) || strings.Contains(strings.ToLower(torrents[i].Name), strings.ToLower(candidate)) {
					// Check if duplicate is already completed
					if torrents[i].PercentDone >= 1.0 {
						h.SendWithFormat(ud.Message.Chat.ID, "duplicated detected, skipping", "add")
					} else {
						h.SendWithFormat(ud.Message.Chat.ID, "duplicated detected (not completed), skipping", "add")
					}
					return
				}
			}
		}
	}

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

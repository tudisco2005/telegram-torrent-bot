package commands

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/tudisco2005/telegram-torrent-bot/commands/helpers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"

	"github.com/pyed/transmission"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	"github.com/tudisco2005/telegram-torrent-bot/utils"
)

// ReceiveTorrent handles torrent file uploads: saves to DEFAULT_TORRENT_LOCATION then adds to Transmission
func ReceiveTorrent(h *handlers.Handler, ud tgbotapi.Update) {
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

	h.Logger.Printf("[DEBUG] ReceiveTorrent: ensuring directory exists: %s", h.DefaultTorrentLocation)
	if err := os.MkdirAll(h.DefaultTorrentLocation, 0755); err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*receiver:* "+err.Error(), "add")
		return
	}

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

	h.Logger.Printf("[DEBUG] ReceiveTorrent: adding to Transmission from file %s", savePath)

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

	_, blocked := helpers.ValidateAddedTorrent(
		h,
		ud.Message.Chat.ID,
		"add",
		added.ID,
		"",
		"Added torrent: metadata not available yet — letting Transmission fetch metadata",
	)
	if blocked {
		return
	}

	if torrent, terr := h.Client.GetTorrent(added.ID); terr == nil {
		if torrent.PercentDone < 1.0 && torrent.Status != transmission.StatusStopped {
			helpers.AddTrackedIDs(h, []int{added.ID})
		}
	} else {
		h.Logger.Printf("[DEBUG] ReceiveTorrent: failed to inspect torrent id=%d for tracking: %v", added.ID, terr)
	}

	safeName := utils.EscapeFileMD(added.Name)
	msg := h.FormatOutputString("add", safeName)
	h.SendWithFormat(ud.Message.Chat.ID, msg, "add")
}

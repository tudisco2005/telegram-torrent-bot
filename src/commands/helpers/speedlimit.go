package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pyed/transmission"
	"github.com/tudisco2005/telegram-torrent-bot/handlers"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// SpeedLimit sets either download or upload limit
func SpeedLimit(h *handlers.Handler, ud tgbotapi.Update, tokens []string, limitType transmission.SpeedLimitType, cmd string) {
	if len(tokens) == 0 {
		uploadLimit, downloadLimit, err := getCurrentLimits(h)
		if err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*"+cmd+":* "+err.Error(), cmd)
			return
		}

		limitMsg := fmt.Sprintf(
			"The current limits are:\n- ↑ Upload: %s\n- ↓ Download: %s",
			formatLimit(uploadLimit),
			formatLimit(downloadLimit),
		)
		h.SendWithFormat(ud.Message.Chat.ID, limitMsg, cmd)
		return
	}

	if strings.EqualFold(tokens[0], "reset") {
		if err := clearLimit(h, limitType); err != nil {
			h.SendWithFormat(ud.Message.Chat.ID, "*"+cmd+":* "+err.Error(), cmd)
			return
		}

		h.SendWithFormat(ud.Message.Chat.ID,
			fmt.Sprintf("*%s:* limit has been removed", cmd), cmd)
		return
	}

	limit, err := strconv.ParseUint(tokens[0], 10, 32)
	if err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "Please, specify the limit as number of kilobytes", cmd)
		return
	}

	if err := applyLimit(h, limitType, uint(limit)); err != nil {
		h.SendWithFormat(ud.Message.Chat.ID, "*"+cmd+":* "+err.Error(), cmd)
		return
	}

	h.SendWithFormat(ud.Message.Chat.ID,
		fmt.Sprintf("*%s:* limit has been successfully changed to %d KB/s", cmd, limit), cmd)
}

func formatLimit(value uint) any {
	if value == 0 {
		return "unlimited"
	}
	return fmt.Sprintf("%d KB/s", value)
}

func getCurrentLimits(h *handlers.Handler) (upload uint, download uint, err error) {
	args, err := sessionCall(h, "session-get", nil)
	if err != nil {
		return 0, 0, err
	}

	upload = uint(readFloat64(args, "speed-limit-up"))
	download = uint(readFloat64(args, "speed-limit-down"))

	uploadEnabled := readBool(args, "speed-limit-up-enabled")
	downloadEnabled := readBool(args, "speed-limit-down-enabled")

	if !uploadEnabled {
		upload = 0
	}
	if !downloadEnabled {
		download = 0
	}

	return upload, download, nil
}

func clearLimit(h *handlers.Handler, limitType transmission.SpeedLimitType) error {
	args := map[string]interface{}{}
	switch limitType {
	case transmission.DownloadLimitType:
		args["speed-limit-down-enabled"] = false
	case transmission.UploadLimitType:
		args["speed-limit-up-enabled"] = false
	default:
		return fmt.Errorf("unable to set limit: invalid limit type")
	}

	_, err := sessionCall(h, "session-set", args)
	return err
}

func applyLimit(h *handlers.Handler, limitType transmission.SpeedLimitType, limit uint) error {
	args := map[string]interface{}{}
	switch limitType {
	case transmission.DownloadLimitType:
		args["speed-limit-down"] = limit
		args["speed-limit-down-enabled"] = true
	case transmission.UploadLimitType:
		args["speed-limit-up"] = limit
		args["speed-limit-up-enabled"] = true
	default:
		return fmt.Errorf("unable to set limit: invalid limit type")
	}

	_, err := sessionCall(h, "session-set", args)
	return err
}

func sessionCall(h *handlers.Handler, method string, args map[string]interface{}) (map[string]interface{}, error) {
	if h.RPCURL == "" {
		return nil, fmt.Errorf("RPC URL is not configured")
	}

	body := map[string]interface{}{"method": method}
	if args != nil {
		body["arguments"] = args
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 20 * time.Second}
	sessionID := ""

	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequest(http.MethodPost, h.RPCURL, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(h.RPCUsername, h.RPCPassword)
		req.Header.Set("Content-Type", "application/json")
		if sessionID != "" {
			req.Header.Set("X-Transmission-Session-Id", sessionID)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusConflict {
			sessionID = resp.Header.Get("X-Transmission-Session-Id")
			resp.Body.Close()
			continue
		}

		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		var out struct {
			Result    string                 `json:"result"`
			Arguments map[string]interface{} `json:"arguments"`
		}

		if err := json.Unmarshal(data, &out); err != nil {
			return nil, err
		}

		if out.Result != "" && out.Result != "success" {
			return nil, fmt.Errorf(out.Result)
		}

		if out.Arguments == nil {
			return map[string]interface{}{}, nil
		}

		return out.Arguments, nil
	}

	return nil, fmt.Errorf("failed to establish Transmission RPC session")
}

func readBool(args map[string]interface{}, key string) bool {
	v, ok := args[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	if !ok {
		return false
	}
	return b
}

func readFloat64(args map[string]interface{}, key string) float64 {
	v, ok := args[key]
	if !ok {
		return 0
	}
	f, ok := v.(float64)
	if !ok {
		return 0
	}
	return f
}

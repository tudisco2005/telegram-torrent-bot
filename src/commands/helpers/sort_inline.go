package helpers

import (
	"fmt"
	"strings"

	"github.com/pyed/transmission"
)

type TorrentSorter func(transmission.Torrents)

func ParseInlineSort(tokens []string) (TorrentSorter, []string, error) {
	if len(tokens) == 0 || strings.ToLower(tokens[0]) != "sort" {
		return nil, tokens, nil
	}

	tokens = tokens[1:]
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("missing sorting method")
	}

	var reversed bool
	if strings.ToLower(tokens[0]) == "rev" {
		reversed = true
		tokens = tokens[1:]
	}

	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("missing sorting method after rev")
	}

	method := strings.ToLower(tokens[0])
	remain := tokens[1:]

	sorter := func(ts transmission.Torrents) {}

	switch method {
	case "id":
		sorter = func(ts transmission.Torrents) { ts.SortID(reversed) }
	case "name":
		sorter = func(ts transmission.Torrents) { ts.SortName(reversed) }
	case "age":
		sorter = func(ts transmission.Torrents) { ts.SortAge(reversed) }
	case "size":
		sorter = func(ts transmission.Torrents) { ts.SortSize(reversed) }
	case "progress":
		sorter = func(ts transmission.Torrents) { ts.SortProgress(reversed) }
	case "downspeed":
		sorter = func(ts transmission.Torrents) { ts.SortDownSpeed(reversed) }
	case "upspeed":
		sorter = func(ts transmission.Torrents) { ts.SortUpSpeed(reversed) }
	case "download":
		sorter = func(ts transmission.Torrents) { ts.SortDownloaded(reversed) }
	case "upload":
		sorter = func(ts transmission.Torrents) { ts.SortUploaded(reversed) }
	case "ratio":
		sorter = func(ts transmission.Torrents) { ts.SortRatio(reversed) }
	default:
		return nil, nil, fmt.Errorf("unknown sorting method")
	}

	return sorter, remain, nil
}

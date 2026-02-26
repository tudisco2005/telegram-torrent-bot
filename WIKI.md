# Telegram Torrent Bot Wiki

This document is the complete reference for this repository: runtime behavior, commands, environment configuration, operational details, and extension points for development.



## 1) What this bot does

This bot controls a Transmission daemon via Telegram.

Core capabilities:

- manage torrents (add/start/stop/check/delete)
- inspect state (list/filter/info/stats/speed/count/trackers)
- set transfer limits
- receive `.torrent` files from Telegram documents
- send startup + completion notifications to known chats
- copy completed downloads to a target directory with progress + duplicate detection
- paginate long responses with inline callback buttons



## 2) Runtime behavior (end-to-end)

## 2.1 Startup flow

On process startup:

1. Parse CLI flags (`src/init.go`).
2. Load `.env` from `.env` then `../.env`, then environment variables (`src/config/env.go`).
3. Validate required config values.
4. Initialize Transmission RPC client.
5. Initialize Telegram bot API + update channel.
6. Load command metadata from JSON.
7. Load known chat IDs from `telegram/chat.json` (or `src/telegram/chat.json` fallback path set).
8. Optionally prune old chat IDs using `REMOVE_ID_OLDER_THAN`.
9. Send startup message to known chats.
10. Start:
   - Telegram update/event loop
   - completion poller loop (every ~30s)

## 2.2 Authorization model

- Only users listed in `MASTER` are authorized.
- Non-master messages and callback queries are ignored.
- Chat IDs are persisted only when the sender is authorized.
- `MASTER` usernames are normalized to lowercase and stripped of `@`.

## 2.3 Message parsing rules

- Private chat: command may be sent with or without `/`.
- Group chat: prefer `/command`.
- Bare message starting with `magnet` or `http` is auto-routed as `add`.
- Unknown command + attached Telegram document is treated as torrent upload flow.
- Empty message with no document is ignored.

## 2.4 Command dispatch model

- Command metadata lives in:
  - `src/telegram/commands.json`
  - `src/telegram/categories.json`
- Dispatcher builds command/alias map at runtime.
- Canonical command name is resolved for formatting/templates.
- `help` and `version` are handled as explicit special cases in dispatcher logic.



## 3) Command reference (full)

All command formats below can generally be used with or without `/` in private chat.

## 3.1 Listing commands

All listing commands in this section use the currently active Transmission sort order (set with `sort` / `so`).
If no custom sort is set, Transmission default ordering is used.

### `list` (`li`, `ls`)
- Purpose: list all torrents.
- Output: one line per torrent with ID + escaped name.

### `plist` (`pls`)
- Purpose: pretty list with progress bars + ETA.
- Output: markdown-rich multi-line block per torrent.
- Default (`plist`): shows only not-complete and not-stopped torrents (active/downloading set).
- `plist all`: shows all torrents (active, stopped, complete).
- `plist stopped` (or `plist paused`): shows paused/stopped and not-complete torrents.
- `plist ?` or `plist help`: shows usage help.
- Optional trailing text filters by name (case-insensitive substring).
- Optional inline sort options are supported; without inline sort, `plist all` is grouped as active -> stopped -> complete.

### `head` (`he`) `[n]`
- Purpose: list first `n` torrents.
- Default behavior: command implementation clamps/bounds list length.
- Supports live updates when enabled.

### `tail` (`ta`) `[n]`
- Purpose: list last `n` torrents.
- Supports live updates when enabled.

### `latest` (`la`) `[n]`
- Purpose: list newest torrents.

### `sort` (`so`) `<method>`
- Purpose: set Transmission sort mode.
- Methods: `id`, `name`, `age`, `size`, `progress`, `downspeed`, `upspeed`, `download`, `upload`, `ratio`.
- Reverse: prefix with `rev` (example: `sort rev size`).
- Help: `sort ?` or `sort help`.

## 3.2 Filtering commands

### `downs` (`dg`)
- List downloading torrents.

### `seeding` (`sd`)
- List seeding torrents.

### `paused` (`pa`)
- List paused torrents.

### `checking` (`ch`)
- List torrents currently verifying.

### `active` (`ac`)
- List active torrents with current up/down rates and ratio.

### `errors` (`er`)
- List torrents with error state.

### `search` (`se`) `<query...>`
- Search torrents by name.

## 3.3 Management commands

### `add` (`ad`) `<magnet|http(s)>`
- Add torrent by magnet/http URL.
- If URL tokens are split by spaces, bot re-joins args before submission.

### `start` (`st`) `<id...|all>`
- Start selected torrents or all.

### `stop` (`sp`) `<id...|all>`
- Stop selected torrents or all.

### `check` (`ck`) `<id...|all>`
- Verify selected torrents or all.

### `del` (`rm`) `<id...>`
- Remove torrent records only (keep data).

### `deldata` `<id...>`
- Remove torrent records and data.

### `move` subcommands
- `move`: show move status for torrents.
- `move all`: copy all not-yet-moved eligible entries.
- `move <id> [id2...]`: copy selected torrent IDs.
- `move <name> [name2...]`: copy by source entry name.
- `move reset`: clear move records file only.
- `move clear`: delete destination content (except skipped/internal entries) and reset records.
- `move ?`: show move help.

Move behavior details:

- Source directory: `DefaultDownloadLocation`, fallback `TRANSMISSION_DOWNLOAD_LOCATION`.
- Destination: `DefaultMoveLocation`, fallback `DEFAULT_MOVE_LOCATION`.
- State file: `../moved.json` relative to destination directory.
- Skips hidden/temp/partial entries (`.*`, `.part`, `.crdownload`).
- Computes SHA1 path hash to detect duplicates already present at destination.
- Sends progress message updates during copy operation.

## 3.4 Information commands

### `info` (`in`) `<id...>`
- Show detailed torrent info (size/progress/speeds/ratio/ETA/tracker list).
- Supports live updates when enabled.

### `trackers` (`tr`)
- List trackers.

### `stats` (`sa`)
- Show current + cumulative Transmission stats.

### `speed` (`ss`)
- Show global download/upload speeds.
- Supports live updates when enabled.

### `count` (`co`)
- Show counts by torrent status categories.

## 3.5 Configuration commands

### `downlimit` (`dl`) `<KB/s>`
- Set global download speed limit.

### `uplimit` (`ul`) `<KB/s>`
- Set global upload speed limit.

## 3.6 General commands

### `help` (`h`, `?`)
- Build and show grouped help from JSON categories + command metadata.

### `version` (`ver`)
- Show Transmission version + bot version (`v1.0.0` in source at time of writing).

### `uptime`
- Show host system uptime.

### `diskusage` (`du`)
- Show used/total usage for configured download location.



## 4) Advanced runtime subsystems

## 4.1 Output formatting model

Each command has optional metadata fields in `commands.json`:

- `output_format`: `markdown` or `plain`
- `output_string`: `fmt.Sprintf`-style template
- `list_output`: apply template per list row when true

At runtime, the handler builds and uses maps keyed by canonical command name:

- `OutputFormatByCommand`
- `OutputStringByCommand`
- `ListOutputByCommand`

Relevant methods:

- `FormatOutputString(...)`
- `SendWithFormat(...)`
- `SendWithPaginationFormat(...)`

## 4.2 Pagination model

Long messages are paginated in handler layer:

- split threshold: ~750 runes/page
- callback data prefix: `pg:`
- buttons: `◀️ Prev`, `Next ▶️`
- session TTL: 5 minutes
- page footer format: `Page X/Y`

Only authorized users can use callback navigation.

## 4.3 Live task cancellation model

For edit-loop style commands, handler uses keyed cancellable tasks:

- `StartLiveTask(key)` cancels any existing task with same key
- prevents overlapping stale loops in same chat/command scope
- global controls:
  - `NO_LIVE=true` disables live updates
  - `UPDATE_MAX_ITERATIONS=0` effectively disables recurring live edits

## 4.4 Completion notification poller

Background poller in Telegram runtime:

- interval: ~30 seconds
- compares currently incomplete torrent IDs vs tracked set
- when previously incomplete now complete: send grouped message
- tracked state is persisted to `track.json`
- chat IDs are reloaded periodically so newly authorized chats receive notifications

## 4.5 Torrent document upload flow

When user sends `.torrent` file:

1. Get Telegram file metadata
2. Download file using bot token URL
3. Ensure `DEFAULT_TORRENT_LOCATION` exists
4. Save file with safe `.torrent` filename
5. Add torrent to Transmission via add-by-file RPC
6. Return add result message

If message is unknown command + document exists, this flow is used as fallback.



## 5) Configuration and environment variables

## 5.1 Resolution and precedence

Effective load order:

1. CLI flags
2. `.env` in current project
3. `.env` in parent directory
4. process environment variables

Important behavior:

- env values only fill fields not already set by flags/defaults.
- `RPC_URL` env value is applied when URL is still default (`http://localhost:9091/transmission/rpc`).

## 5.2 Required variables

- `TOKEN` or `TT_BOTT`: Telegram bot token
- `MASTER`: comma-separated usernames (without `@` preferred)
- `USERNAME` or `TR_AUTH`: Transmission username
- `PASSWORD`: Transmission password
- `RPC_URL`: Transmission RPC endpoint

## 5.3 Optional variables

- `BOT_LOGFILE`: log file path
- `DEFAULT_TORRENT_LOCATION`: where uploaded `.torrent` files are saved
- `DEFAULT_DOWNLOAD_LOCATION`: download/source directory
- `TRANSMISSION_DOWNLOAD_LOCATION`: alternate/fallback download source variable
- `DEFAULT_MOVE_LOCATION`: destination directory used by `move`
- `NO_LIVE`: `true|false`
- `VERBOSE`: `1` or `true`
- `REMOVE_ID_OLDER_THAN`: seconds, prune old chat IDs at startup
- `UPDATE_MAX_ITERATIONS`: integer `>= 0`

## 5.4 CLI flags

Primary flags (from `src/init.go`):

- `-token`
- `-master` (repeatable)
- `-url`
- `-username`
- `-password`
- `-logfile`
- `-no-live`
- `-verbose`

## 5.5 Validation and common pitfalls

- startup fails on missing required config values.
- `UPDATE_MAX_ITERATIONS` must be `>= 0`.
- `DEFAULT_TORRENT_LOCATION` must exist/be writable for file upload flow.
- `move` command requires both source and destination configuration.
- `start.sh` requires `transmission-remote` in `PATH`.



## 6) Files, data, and persistence

## 6.1 Main persistent files

- `src/telegram/chat.json`: known chat IDs + timestamp
- `src/telegram/track.json`: tracked incomplete torrent IDs for completion notifications
- `data/moved.json` (actual location derived from destination parent): move history

## 6.2 Runtime path resolution

Several files are loaded using fallback path candidates (`resolveExistingPath`), allowing run from different working directories.

## 6.3 Log output

- default logger writes to stdout
- if `BOT_LOGFILE` or `-logfile` set, logger writes to both stdout and file



## 7) Project architecture for developers

## 7.1 Key modules

- `src/main.go`: app bootstrap, defaults, signal-aware shutdown context
- `src/init.go`: flags/env load, Transmission and Telegram initialization
- `src/telegram/telegram.go`: update loop, authorization, command dispatch, startup + completion logic
- `src/handlers/`: output formatting, pagination, live task controls
- `src/commands/`: individual command implementations
- `src/utils/`: markdown escaping, persistence helpers, sending/progress helpers

## 7.2 Add a new command (clean workflow)

1. Create implementation in `src/commands/<name>.go`.
2. Register canonical command in `handlerMap` in `src/telegram/telegram.go`.
3. Add command metadata to `src/telegram/commands.json`:
   - `name`, `aliases`, `command_category`, `description`
   - optional `example`, `output_format`, `output_string`, `list_output`
4. Ensure category exists in `src/telegram/categories.json`.
5. Build and run, test from Telegram chat.

## 7.3 Compatibility and style conventions

- keep command names and aliases synchronized between code and JSON metadata.
- preserve markdown escaping for user/torrent/file-visible text.
- preserve existing message style (`*command:* ...`) for consistency.
- avoid introducing new persistence files without documenting path and lifecycle.
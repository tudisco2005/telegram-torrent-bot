# Telegram Torrent Bot Wiki

This wiki is a practical guide for:

- using the bot (commands + runtime behavior)
- configuring environment variables
- further development (logic, structure, extension points)


## 1) Using the Bot

## 1.1 Authorization and message handling behavior

- Only users in `MASTER` are allowed to execute commands.
- Non-master messages are ignored (no command execution).
- Chat IDs are persisted in `src/telegram/chat.json` only when the sender is authorized.
- At startup, the bot loads known chat IDs and sends a startup message.
- Bare `magnet...` or `http(s)...` messages are auto-treated as `add` command.
- Uploaded Telegram documents are treated as torrent uploads when command lookup fails.

Notes:

- In private chats, commands can be sent with or without `/`.
- In group chats, use `/command` format.

## 1.2 Help/commands metadata model

Command metadata is defined in:

- `src/telegram/commands.json` (aliases, descriptions, output format/template)
- `src/telegram/categories.json` (help grouping/order)

The dispatcher builds a name+alias map at runtime. `help` and `version` are handled as special cases.

## 1.3 Command behavior reference

### List/filter commands

- `list`, `plist`, `head`, `tail`
- `downs`, `seeding`, `paused`, `checking`, `active`, `errors`
- `search`, `latest`, `trackers`, `count`

Behavior:

- Most list outputs are markdown-formatted via templates in `commands.json`.
- Large outputs are paginated with inline buttons (`Prev`/`Next`) and temporary pagination sessions.

### Management commands

- `add`: add URL/magnet links to Transmission.
- `stop`, `start`, `check`: operate by IDs or `all` where supported.
- `del`: remove torrent records only.
- `deldata`: remove torrent + downloaded data.
- `move`: copy/list/reset/clear move state and destination content.

### Information/general/config commands

- `info`, `stats`, `speed`, `diskusage`, `uptime`, `version`
- `downlimit`, `uplimit`
- `help`

## 1.4 Live update behavior

Some commands can live-edit the same Telegram message (for progress-like output), including:

- `speed`, `info`, `head`, `tail`, `plist`, `move`  (and any command implementing edit loops)

Controls:

- `NO_LIVE=true` disables live edits.
- `UPDATE_MAX_ITERATIONS=0` also disables live updates.
- If enabled, updates run on internal interval/duration and can be capped by `UPDATE_MAX_ITERATIONS`.
- A per-command chat key is used to cancel previous live tasks when a new one starts.

## 1.5 Torrent completion notifications

Background poller behavior:

- Polls torrents periodically.
- Compares current incomplete IDs vs tracked incomplete IDs.
- Sends grouped "Downloads complete" messages to known chat IDs.
- Persists tracking state to `src/telegram/track.json`.

## 1.6 Torrent file upload behavior

When a `.torrent` file is sent as Telegram document:

1. File metadata is fetched from Telegram API.
2. File is downloaded using bot token.
3. Saved to `DEFAULT_TORRENT_LOCATION`.
4. Added via Transmission add-by-file RPC.
5. Bot responds with add result.

## 1.7 Move workflow details

`move` command uses:

- source: `TRANSMISSION_DONWNLOAD_LOCATION`
- destination: `DEFAULT_MOVE_LOCATION`
- state file: `moved.json` stored near destination (`../moved.json`)

Subcommands:

- `move` => list status of torrents vs source/destination state
- `move all` => copy all not-yet-moved entries
- `move <id> [id...]` => copy selected torrent IDs (or names)
- `move reset` => clear only move records
- `move clear` => clear destination content + reset records
- `move ?` => help

Behavior highlights:

- Skips temp/partial files.
- Uses SHA1 hash-based duplicate detection in destination.
- Shows progress updates while copying.

---

## 2) Environment Variables

## 2.1 Resolution order

Configuration is loaded in this effective order:

1. CLI flags
2. `.env` (`.env`, then `../.env`)
3. process environment variables

Important: environment values only fill fields that are still unset/default after CLI parsing.

## 2.2 Required variables

- `TOKEN` (or `TT_BOTT`): Telegram bot token
- `MASTER`: comma-separated Telegram usernames (without `@` preferred)
- `USERNAME` (or `TR_AUTH`): Transmission RPC username
- `PASSWORD`: Transmission RPC password
- `RPC_URL`: Transmission RPC endpoint

## 2.3 Optional variables

- `BOT_LOGFILE`: log output file path
- `DEFAULT_TORRENT_LOCATION`: where uploaded `.torrent` files are saved
- `DEFAULT_DOWNLOAD_LOCATION`: source dir for move workflow
- `DEFAULT_MOVE_LOCATION`: destination dir for move workflow
- `NO_LIVE`: boolean (`true/false`) to disable live edits
- `VERBOSE`: `1` or `true` enables debug logs
- `REMOVE_ID_OLDER_THAN`: seconds; prunes stale chat IDs at startup
- `UPDATE_MAX_ITERATIONS`: integer `>= 0`
  - `0` => disable live updates
  - `>0` => cap live edit iterations per message
- `TRANSMISSION_DONWNLOAD_LOCATION`: legacy fallback variable (spelling preserved intentionally)

## 2.4 Validation and common pitfalls

- Bot startup fails if required values are missing.
- `UPDATE_MAX_ITERATIONS` must be `>= 0`.
- `MASTER` names are normalized to lowercase and stripped of `@`.
- `RPC_URL` from env only overrides default URL when URL is still default/unset from flags.

---

## 3) Further Developing the Bot

## 3.1 High-level architecture

Main flow:

- `src/main.go`: app bootstrap + signal handling
- `src/init.go`: flags + env loading + client init
- `src/telegram/telegram.go`: event loop, auth, dispatch, startup notifications, completion poller
- `src/handlers/`: shared runtime behavior (formatting, pagination, live-task management)
- `src/commands/`: command implementations
- `src/utils/`: message sending, chat/tracked persistence, markdown/progress helpers

## 3.2 Dispatch and extension model

To add a new command cleanly:

1. Implement command logic in `src/commands/<name>.go`.
2. Add entry in `handlerMap` in `src/telegram/telegram.go`.
3. Add metadata in `src/telegram/commands.json`:
   - `name`, `aliases`, `command_category`, `description`
   - `output_format`, `output_string`, `list_output`
4. (Optional) add category in `src/telegram/categories.json`.
5. Rebuild and test in Telegram chat.

Tip: prefer reusing helper patterns from existing commands (`helpers/` package).

## 3.3 Output formatting system

Each command can be configured without code changes for output style:

- `output_format`: markdown/plain
- `output_string`: `fmt.Sprintf` template
- `list_output`: whether template is applied per list item

Runtime maps are built from JSON and consumed through:

- `h.FormatOutputString(...)`
- `h.SendWithFormat(...)`
- `h.SendWithPaginationFormat(...)`

## 3.4 Pagination subsystem

Pagination is handler-driven:

- long text is split into pages
- inline callback buttons navigate pages
- sessions are ephemeral (TTL-based)
- callback data prefix: `pg:`

Useful when adding commands that can produce long multi-line output.

## 3.5 Live task cancellation model

For commands with repeated edits:

- use `StartLiveTask(key)` to ensure only one live loop per key
- previous task with same key is cancelled
- key examples: `speed:<chat>`, `info:<chat>:<torrentID>`

This prevents overlapping edit loops and stale updates.

## 3.6 Persistence model

- `src/telegram/chat.json`: known chat IDs + timestamp
- `src/telegram/track.json`: tracked incomplete torrents for completion notices
- `data/moved.json` (path relative to move destination): move/copy history

Write path behavior is already safe/atomic for tracked IDs via utils helper.

## 3.7 Practical development workflow

- Install deps in `src/`: `go mod download`
- Build binary: `go build -o ../bin/telegram-torrent-bot`
- Run: `./start.sh` or `./bin/telegram-torrent-bot`
- Enable verbose mode while developing: `VERBOSE=1` or `-verbose`

## 3.8 Conventions to keep

- Keep command names/aliases in sync between code and JSON.
- Preserve markdown escaping for torrent/file names.
- Reuse existing error style (`*command:* error`) for consistency.
- Keep new persistence files in predictable paths and document them.


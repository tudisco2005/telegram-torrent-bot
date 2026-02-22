# Telegram Torrent Bot (Transmission)

A Telegram bot to control a Transmission daemon from chat.

> This project is based on the original [transmission-telegram](https://github.com/pyed/transmission-telegram/) with some personal improvements

It supports listing/filtering torrents, start/stop/check/delete actions, adding torrents (URL/magnet and uploaded `.torrent` files), speed limits, disk usage, move/copy workflows, and automatic completion notifications.

## Features

- Restricts command execution to configured Telegram master usernames.
- Uses Transmission RPC (`github.com/pyed/transmission`) for torrent management.
- Loads command metadata (aliases, help text, output formatting) from JSON.
- Supports live-updating messages for status commands (editable Telegram messages).
- Persists known chat IDs for startup/completion notifications.
- Persists completed-notification tracking to avoid duplicate alerts after restart.
- Supports uploaded `.torrent` files from Telegram documents.

## Project Layout

- `src/main.go`: application entry point and wiring.
- `src/init.go`: CLI flags + environment loading + client initialization.
- `src/telegram/telegram.go`: Telegram event loop, auth checks, dispatching, completion poller.
- `src/commands/*.go`: command implementations.
- `src/config/env.go`: `.env`/environment parsing and validation.
- `src/telegram/commands.json`: command definitions used for dispatch/help/format.
- `src/telegram/categories.json`: help categories.
- `start.sh`: convenience startup script (build-if-needed + run).

## Requirements

- Linux host (or compatible POSIX shell environment), tested on:
    - Ubuntu 24.10 
    - Ubuntu 24.04 LTS
- Go `1.23.2+`
- Running Transmission daemon with RPC enabled
- `transmission-remote` installed and available in `PATH` (required by `start.sh`)
- Telegram bot token from BotFather

## Installation

### 1) Clone and enter the project

```bash
git clone https://github.com/tudisco2005/telegram-torrent-bot.git
cd telegram-torrent-bot
```

### 2) Install dependencies

```bash
cd src
go mod download
cd ..
```

### 3) Ensure required directories exist

```bash
mkdir -p bin data/downloads data/torrents log
```

## Configuration

The app reads values from:

1. CLI flags
2. `.env` file (`.env` or `../.env` from the process working directory)
3. Environment variables

CLI values take precedence over environment when already set.

### Required environment variables

> Current code validation requires all of these:

- `TOKEN` (or `TT_BOTT`) — Telegram bot token
- `MASTER` — one or more Telegram usernames (comma-separated), without `@` preferred
- `USERNAME` (or `TR_AUTH`) — Transmission RPC username
- `PASSWORD` — Transmission RPC password
- `RPC_URL` — Transmission RPC URL (e.g. `http://localhost:9091/transmission/rpc`)
- `UPDATE_MAX_ITERATIONS` — integer `> 0` (required by current validation)

### Optional environment variables

- `BOT_LOGFILE` — log file path
- `DEFAULT_TORRENT_LOCATION` — where uploaded `.torrent` files are stored before adding
- `DEFAULT_DOWNLOAD_LOCATION` — download directory used by some commands
- `DEFAULT_MOVE_LOCATION` — destination for `/move` copy operations
- `VERBOSE` — `1/true` enables debug logs
- `REMOVE_ID_OLDER_THAN` — seconds; prunes stale chat IDs at startup
- `TRANSMISSION_DONWNLOAD_LOCATION` — legacy/fallback variable (note the spelling used by the code)

## Example `.env`

```env
# Telegram
TOKEN=123456789:ABCDEF_your_bot_token
MASTER=your_username,another_username

# Transmission RPC
RPC_URL=http://127.0.0.1:9091/transmission/rpc
USERNAME=transmission
PASSWORD=transmission

# Runtime behavior
UPDATE_MAX_ITERATIONS=10
VERBOSE=0

# Paths
DEFAULT_TORRENT_LOCATION=./data/torrents
DEFAULT_DOWNLOAD_LOCATION=./data/downloads
DEFAULT_MOVE_LOCATION=./data/moved

# Optional
BOT_LOGFILE=./log/bot.log
REMOVE_ID_OLDER_THAN=2592000
```

## Running

### Option A: Use startup script

```bash
./start.sh
```

With CLI overrides:

```bash
./start.sh -token="..." -master="@your_username" -url="http://127.0.0.1:9091/transmission/rpc" -username="..." -password="..."
```

### Option B: Build and run manually

```bash
cd src
go build -o ../bin/telegram-torrent-bot
cd ..
./bin/telegram-torrent-bot
```

## CLI Flags

- `-token` Telegram bot token
- `-master` master username (repeatable)
- `-url` Transmission RPC URL (default: `http://localhost:9091/transmission/rpc`)
- `-username` Transmission RPC username
- `-password` Transmission RPC password
- `-logfile` log output file
- `-no-live` disable live edit updates
- `-verbose` enable debug logging

## Command Reference

The bot accepts commands with or without leading `/` in private chats. In groups, using `/` is recommended.

It also treats a bare magnet/http URL message as an implicit `add` command.

| Command | Aliases | Category | Description | Example |
|---|---|---|---|---|
| `/list` | `/li`, `/ls` | list | Lists all the torrents | - |
| `/plist` | `/pls` | list | Pretty list with progress bars | - |
| `/head` | `/he` | list | Lists the first n torrents | `/head 10` |
| `/tail` | `/ta` | list | Lists the last n torrents | `/tail 10` |
| `/downs` | `/dg` | filter | Lists downloading torrents | - |
| `/seeding` | `/sd` | filter | Lists seeding torrents | - |
| `/paused` | `/pa` | filter | Lists paused torrents | - |
| `/checking` | `/ch` | filter | Lists torrents being verified | - |
| `/active` | `/ac` | filter | Lists actively uploading/downloading torrents | - |
| `/errors` | `/er` | filter | Lists torrents with errors | - |
| `/sort` | `/so` | list | Manipulate sorting | `/sort [option]` |
| `/trackers` | `/tr` | information | Lists all trackers | - |
| `/add` | `/ad` | management | Add URLs or magnets | `/add <magnet-link> or /add <http-url>` |
| `/search` | `/se` | filter | Search torrents by name | `/search <query>` |
| `/latest` | `/la` | list | Lists newest torrents | `/latest 5` |
| `/info` | `/in` | information | Get info about torrents | `/info <id1> <id2>` |
| `/stop` | `/sp` | management | Stop torrents | `/stop <id1> <id2> or /stop all` |
| `/start` | `/st` | management | Start torrents | `/start <id1> <id2> or /start all` |
| `/check` | `/ck` | management | Verify torrents | `/check <id1> <id2> or /check all` |
| `/del` | `/rm` | management | Delete only torrents | `/del <id1> <id2>` |
| `/deldata` | - | management | Delete torrents and data | `/deldata <id1> <id2>` |
| `/move` | - | management | Copy/list/clear downloads using move workflow | `/move ?` |
| `/stats` | `/sa` | information | Show transmission stats | - |
| `/uptime` | - | general | Show system uptime | - |
| `/diskusage` | `/du` | general | Show used disk space for download directory | - |
| `/downlimit` | `/dl` | configuration | Set download speed limit | `/downlimit 1024` |
| `/uplimit` | `/ul` | configuration | Set upload speed limit | `/uplimit 512` |
| `/speed` | `/ss` | information | Show upload/download speeds | - |
| `/count` | `/co` | information | Show torrents count per status | - |
| `/help` | `/h`, `/?` | general | Show help message | - |
| `/version` | `/ver` | general | Show version numbers | - |

### `/sort` options

- `id`, `name`, `age`, `size`, `progress`, `downspeed`, `upspeed`, `download`, `upload`, `ratio`
- Prefix with `rev` for reverse order (e.g. `sort rev size`)

### `/move` subcommands

- `move` — show per-torrent move status
- `move all` — copy all not-yet-moved entries
- `move <id> [id2 ...]` — copy by torrent IDs
- `move reset` — clear move records
- `move clear` — remove entries in destination and clear move records
- `move ?` — show move help

## Authorization Model

- Only users listed in `MASTER` can execute commands.
- Incoming messages from non-master users are ignored.
- Authorized chats are persisted in `src/telegram/chat.json` with timestamps.

## Notifications and Persistence

- On startup, bot sends an online message to known chat IDs.
- A background poller checks torrent completion and sends grouped completion messages.
- Completed-notification state is persisted to `src/telegram/completed.json`.
- Move operation state is persisted in `moved.json` (near `DEFAULT_MOVE_LOCATION`).

## Logging

- Default logs go to stdout.
- If `BOT_LOGFILE` or `-logfile` is set, logs are written to both stdout and file.
- `VERBOSE=true` or `-verbose` enables debug-level operational logs.

## Troubleshooting

### Bot exits on startup with config errors

Check required env values:

- `TOKEN/TT_BOTT`
- `MASTER`
- `USERNAME/TR_AUTH`
- `PASSWORD`
- `RPC_URL`
- `UPDATE_MAX_ITERATIONS` (must be set and greater than `0` in current code)

### Transmission RPC connection fails

- Verify `RPC_URL`, `USERNAME`, and `PASSWORD`.
- Ensure Transmission daemon is running and RPC is enabled.
- Confirm host/container/firewall networking if remote.

### `start.sh` says `transmission-remote` missing

Install transmission CLI tools for your distribution and ensure `transmission-remote` is in `PATH`.

### Uploaded `.torrent` files are rejected

- Ensure `DEFAULT_TORRENT_LOCATION` exists or is writable.
- Verify Telegram bot has access to fetch files and network egress is allowed.

## Development

Build:

```bash
cd src
go build -o ../bin/telegram-torrent-bot
```

Run with logs:

```bash
cd ..
./start.sh -verbose
```

## Version

Current bot constant in source: `v1.0.0`.


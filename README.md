# Telegram Torrent Bot (Transmission)

A Telegram bot to control a Transmission daemon from chat.

> This project is based on the original [transmission-telegram](https://github.com/pyed/transmission-telegram/) with some personal improvements

It supports listing/filtering torrents, start/stop/check/delete actions, adding torrents (URL/magnet and uploaded `.torrent` files), speed limits, disk usage, move/copy workflows, and automatic completion notifications.

### [Demo](https://youtube.com/shorts/JWeyqdnZFio?feature=share)

## Wiki

- See [`WIKI.md`](./WIKI.md) for detailed usage, environment variables, and development internals.

## Features

### Previous features (from [pyed/transmission-telegram](https://github.com/pyed/transmission-telegram/))

- Telegram command interface for Transmission RPC torrent management.
- Listing and filtering torrents by status, plus detailed torrent info.
- Torrent lifecycle actions (start, stop, check, delete).
- Add torrents from magnet links and HTTP(S) URLs.
- Speed/statistics utilities (upload/download speed, limits, tracker and status views).
- Access control via configured Telegram master usernames.

### New features in this fork

- Add torrents by uploading `.torrent` files directly in Telegram.
- Move/copy workflow commands for completed downloads (`/move`, `all`, `reset`, `clear`).
- Persist known chat IDs for startup and completion notifications.
- Persist completed-notification tracking to prevent duplicates after restart.
- Live-updating Telegram messages for status-style commands (editable message output).
- JSON-driven command metadata (aliases, help text, output formatting).
- New commands

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


## Command Reference

The bot accepts commands with or without leading `/` in private chats. In groups, using `/` is recommended.

It also treats a bare magnet/http URL message as an implicit `add` command.

| Command | Aliases | Category | Description | Example |
|---|---|---|---|---|
| `/list` | `/li`, `/ls` | list | Lists all the torrents | - |
| `/plist` | `/pls` | list | Pretty list with progress bars (default: active/downloading only) | `/plist all` |
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

### `/plist` behavior

- `plist` or `pls` — list only not-complete and not-stopped torrents
- `plist all` — list all torrents (active, stopped, complete)
- `plist stopped` or `plist paused` — list stopped/paused and not-complete torrents
- `plist ?` or `plist help` — show usage help
- `plist [mode] <name filter>` — optional case-insensitive name filter
- `plist all` without inline sort groups by state: active -> stopped -> complete

### `/move` subcommands

- `move` — show per-torrent move status
- `move all` — copy all not-yet-moved entries
- `move <id> [id2 ...]` — copy by torrent IDs
- `move reset` — clear move records
- `move clear` — remove entries in destination and clear move records
- `move ?` — show move help

## Version

Current bot constant in source: `v1.0.0`.


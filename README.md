# Telegram Torrent Bot

this project is based on the original (transmission-telegram)[https://github.com/pyed/transmission-telegram/] with some improvement such as:
- clear messages
- detect duplicate torrent
- fixed some bugs 
- code divided in separeted files for better further developing

new feature:
- move.sh: after a torrent is completed is moved to a ./data/copleted folder inside the bot folder and make a link so we can continue seeding, when /deldata only the link file is deleted
- save .torrent files in ./data/torrents folder inside the bot folder
- verbose logging of events

### demo video:


---
## Install

### Prerequisite

install transmission

sudo add-apt-repository ppa:transmissionbt/ppa
sudo apt-get update
sudo apt-get install transmission-cli transmission-common transmission-daemon

install go

sudo apt update
sudo apt install golang-go
go version

### install the bot
https://github.com/tudisco2005/telegram-torrent-bot.git
cd ./telegram-torrent-bot/src
go init telegram-torrent-bot
go get
chmod +x start.sh

edit the transsmission config file
sudo service transmission-daemon stop
sudo nano /var/lib/transmission-daemon/info/settings.json

```json
    "script-torrent-done-enabled": true,
    "script-torrent-done-filename": "/home/<YOUR_USER>/MyTorrentBot/bot/move.sh",
```
sudo service transmission-daemon start

### fix premission

if you donwload the repository in the home folder follow below, if not you need to change the paths

chmod o+x /home/<YOUR_USER>/MyTorrentBot
chmod -R o+rx /home/<YOUR_USER>/MyTorrentBot/bot
sudo chgrp debian-transmission /home/<YOUR_USER>/MyTorrentBot/bot/data/completed/ 
sudo chmod g+w /home/<YOUR_USER>/MyTorrentBot/bot/data/completed/

sudo chgrp debian-transmission /home/dipi/MyTorrentBot/bot/log/
sudo chmod g+w /home/dipi/MyTorrentBot/bot/log/

### setup the env variable
cp .env.example .env

fill the .env file

### Configure the .env file

#### bot env
get a telegram token bot:
1. open telegram
2. search botfather
3. create a bot follwing the requests
4. copy the bot token
5. paste `TOKEN=<PASTE_YOUR_TOKEN_HERE>` #wihout the `<>`

`MASTER=@<TELEGRAM_USER_NAME>` # more masters are possible following this MASTER=user1,user2,user3

#### trasmission env
if the trasmission config is default, you can use
RPC_URL=http://localhost:9091/transmission/rpc
USERNAME=transmission
PASSWORD=transmission

### Run the bot

./start.sh

---
## For Developing
file structure

for saving location of torrent in move.sh change DESTINAZIONE
warning you need to give transmission the permission

other env paramether:
DEFAULT_TORRENT_LOCATION=
LOGFILE=
VERBOSE=0

### to do
- [ ] dockerize
- [ ] clear the code
- [ ] custom buttons
- [ ] notify user when donwnload is complete
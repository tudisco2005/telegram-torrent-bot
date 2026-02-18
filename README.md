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

tested on:
- Ubuntu 24.10
- Ubuntu 24.04 LTS

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
chmod +x start.sh
cd ./telegram-torrent-bot/src
go init telegram-torrent-bot
go get
cd ..

edit the transsmission config file
sudo service transmission-daemon stop
sudo nano /var/lib/transmission-daemon/info/settings.json

```json
    "script-torrent-done-enabled": true,
    "script-torrent-done-filename": "/home/<YOUR_USER>/bot/move.sh",
```
sudo service transmission-daemon start

note: if look stuck or crashes after a bit
systemctl edit --full transmission-daemon.service
edit Type=notify to Type=simple
sudo service transmission-daemon reload
sudo service transmission-daemon start

### make folders

mkdir ./log
mkdir ./data
mkdir ./data/torrents
mkdir ./data/downloads

### fix move.sh
replace

DESTINAZIONE="/home/<DESTINATION_USER>/bot/data/downloads"
LOGFILE="/home/dipi/<DESTINATION_USER>/log/transmission-move.log"

not using $USER because need to be moved to another user home folder

### fix premission

if you donwload the repository in the home folder follow below, if not you need to change the paths

chmod o+x /home/$USER/
chmod -R o+rx /home/$USER/bot
sudo chgrp debian-transmission /home/$USER/bot/data/downloads/ 
sudo chmod g+w /home/$USER/bot/data/downloads/

sudo chgrp debian-transmission /home/$USER/bot/log
sudo chmod g+w /home/$USER/bot/log

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
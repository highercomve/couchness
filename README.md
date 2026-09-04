Couchness is a simple CLI tools to update your shows
=====================================

couchness maintain your show library update by using transmission-remote to download torrents. Therefore you will need transmission-deamon and transmission-remote.

The first time you run any command will create a configuration file on

`~/.couchness/configuration/couchness.json`

The first run you can pass environment variables to set in that configuration file.

```
COUCHNESS_SHOWS_DIR
COUCHNESS_MOVIES_DIR
COUCHNESS_OMDB_API_KEY
COUCHNESS_TRANSMISSION_AUTH (default: transmission:transmission)
COUCHNESS_TRANSMISSION_HOST (default: localhost)
COUCHNESS_TRANSMISSION_PORT (default: 9091)
```
Here you can get you OMDB API key

www.omdbapi.com/apikey.aspx

## Install 

Important you need to have install transmission, because couchness is going to search and mantain your shows or movies, but uses transmission as tools to download the torrents.

**Latest Release**

```bash
wget https://raw.githubusercontent.com/highercomve/couchness/master/install.sh
bash install.sh 
```

**Specific version**

```bash
wget https://raw.githubusercontent.com/highercomve/couchness/master/install.sh
bash install.sh v0.0.1
```

## How to init your library

Couchness scan will read your media library an ask you to select the show names on IMDb (in order to get the IMDB_ID)
After this initial step all the series are going to be in follow mode "latest"

That means will download the latest episode on the next run of `update-all`

```bash
COUCHNESS_SHOWS_DIR=/where/your/shows/are COUCHNESS_OMDB_API_KEY=XXXXXXX couchness scan -i -r
```

### How to update your library

```bash
couchness update-all
```

## Telegram integration

Create a bot with [@BotFather](https://t.me/BotFather), then expose its token to Couchness. The token is never stored in the Couchness database.

```bash
export COUCHNESS_TELEGRAM_BOT_TOKEN="your-bot-token"
couchness telegram setup --owner-id 123456789
couchness telegram run
```

If you do not know your numeric Telegram user ID, omit `--owner-id`, run the bot, and send `/start`. The bot replies with your ID. Authorize it from another terminal:

```bash
couchness telegram users add 123456789 --role owner
```

Manage access from the CLI:

```bash
couchness telegram status
couchness telegram test
couchness telegram users list
couchness telegram users add 987654321 --role user
couchness telegram users remove 987654321
couchness telegram disable
```

Authorized bot commands:

```text
/shows
/movies
/show <show_id>
/add_show <title>
/add_movie <title>
/remove_show [show_id]
/update <show_id>
/download <show_id>
/update_all
/users
/invite [user|viewer|admin]
/revoke <telegram_user_id>
```

Owners and admins can create single-use invite links from the bot. Invites expire after 10 minutes. Only owners can invite or revoke admins; owner accounts are managed through the CLI. The bot accepts commands only in private chats.

`/add_show` provides a guided setup with OMDb search-result buttons, follow-mode and resolution choices, confirmation, and optional immediate download. Owners and admins may use it. Send `/cancel` at any time to close an active setup.

`/add_movie` lets owners and admins search OMDb, choose a torrent, confirm the download, and add the queued movie to the movies list. `/movies` lists queued movies. Send `/cancel` at any time to close an active setup.

`/remove_show` lets owners and admins choose a tracked show, review a clear warning, and remove only its Couchness record. Media files and Transmission torrents remain untouched, and later scans will not re-add the show. Running `/add_show` for it again removes that exclusion.

For a combined Couchness Web, Telegram, Transmission, and Plex deployment, copy `deploy/telegram/couchness.env.example`, fill its values, and run:

```bash
docker compose --env-file couchness.env -f deploy/telegram/compose.yaml up -d --build
```

Plex stores its configuration under `${COUCHNESS_DATA_DIR}/plex` and mounts the shared media directory read-only at `/media`. After startup, open `http://<host>:32400/web`, claim the server, and add `/media/shows` and `/media/movies` as libraries. `PLEX_CLAIM` may be set to a temporary claim token for automatic first-run claiming.

Couchness Web listens on `http://<host>:8085`. The Compose deployment protects it with HTTP Basic Auth using the configured Transmission username and password.

## Movies

`couchness movies download` searches YTS and The Pirate Bay (services `yts` and `tpb`) for torrents
matching the movie's IMDb ID. RARBG shut down in 2023, so it is no longer used as a movie source.

## Web UI

Couchness ships a small web interface (server-rendered HTML + htmx) to scan the library, update shows,
download the latest episode, identify shows on IMDb, and search/queue movies on transmission.

```bash
couchness web run                      # listens on :8085
couchness web run --addr 127.0.0.1:9000 --auth admin:secret
```

`COUCHNESS_WEB_ADDR` and `COUCHNESS_WEB_AUTH` can replace the flags. Pages: `/shows`, `/movies`, `/jobs`.

Install it as a systemd service (system unit needs root; `--user` installs into `~/.config/systemd/user`):

```bash
sudo couchness web install --auth admin:secret --env COUCHNESS_OMDB_API_KEY=xxx
couchness web install --user --print   # only print the unit
couchness web status
couchness web uninstall
```

When run through `sudo` without `--config-dir`, the unit points at the invoking user's `~/.couchness`
and runs as that user (`--run-as`). Use `--env-file` to keep secrets out of the unit.

### Help

```bash
couchness - couchness is an automatic tool to follow and download show using RSS or eztv

USAGE:
   couchness [global options] command [command options] [arguments...]

VERSION:
   0.3.0

AUTHOR:
   Sergio Marin

COMMANDS:
   add, a              add SHOW_NAME FOLDER
   scan, s             scan
   download, d         download SHOW_ID
   update-all, ua      update all your shows
   migrate, m          Migrate shows from monoservice to multiservice
   update, u           update one show using showID
   shows               show
   show                show <SHOW_ID>
   add-shows-dir, asd  add-shows-dir <directory>
   download-ep, de     download SHOW_ID EPISODE maximun_search(optional)
   disable             disable <SHOW_ID>
   movies              movies
   help, h             Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config-dir value  
   --help, -h          show help
   --version, -v       print the version
```

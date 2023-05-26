Couchness is a simple CLI tools to update your shows
=====================================

couchness maintain your show library update by using transmission-remote to download torrents. Therefore you will need transmission-deamon and transmission-remote.

The first time you run any command will create a configuration file on

`~/.couchness/configuration/couchness.json`

The first run you can pass environment variables to set in that configuration file.

```
COUCHNESS_MEDIA_DIR
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
COUCHNESS_MEDIA_DIR=/where/your/shows/are COUCHNESS_OMDB_API_KEY=XXXXXXX couchness scan -i -r
```

### How to update your library

```bash
couchness update-all
```

### Help

```bash
couchness - couchness is an automatic tool to follow and download show using RSS or eztv

USAGE:
   couchness [global options] command [command options] [arguments...]

VERSION:
   0.2.1

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

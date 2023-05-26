#!/bin/bash
set -e

install_location=${1:-"/usr/bin/couchness"}

case $(uname -m | sed 'y/ABCDEFGHIJKLMNOPQRSTUVWXYZ/abcdefghijklmnopqrstuvwxyz/') in
  x86_64)
    ARCH=amd64
    ;;
  *386*)
    ARCH=386
    ;;
  aarch64)
    ARCH=arm
    ;;
  arm*)
    ARCH=arm
    ;;
  *)
    echo "Architecture not supported. only supported amd64, 386 and arm"
    exit 0
    ;;
esac

case $(uname | sed 'y/ABCDEFGHIJKLMNOPQRSTUVWXYZ/abcdefghijklmnopqrstuvwxyz/') in
  linux*)
    OS=linux
    ;;
  darwin*)
    OS=darwin
    ;;
  msys*)
    OS=windows
    ;;
  *)
    echo "OS not supported"
    exit 0
    ;;
esac


PLATFORM=($OS"_"$ARCH)

echo "Downloading couchness build for $PLATFORM to $install_location"

version=`curl -s -L 'https://api.github.com/repos/highercomve/couchness/releases' -H 'Accept: application/vnd.github.v3+json' -H 'Accept-Language: en-US,en;q=0.5' | jq --arg "PLATFORM" "$PLATFORM" '[.[] | select(.prerelease == false)][0].assets | (.[] | select(.name | contains("'$PLATFORM'.zip")))'`
downloadurl=`echo "$version" | jq -r .browser_download_url`
versionnumber=`echo "$version" | jq -r .name`

installed_bin=`whereis couchness | awk '{print $2}'`
if [ "$installed_bin" != "" ]; then
  echo
  echo "You have couchness installed"
  $installed_bin -v
fi

echo ""	
echo "Do you what to download $downloadurl [yes/y]"

read download

if [ "$download" != "yes" ] && [ "$download" != y ]; then
	exit 0
fi

echo "Downloading latest version from: $downloadurl"

tempdir=`mktemp -d`
curl -L "$downloadurl" -o "$tempdir/couchness.zip"
cd $tempdir
unzip $tempdir/couchness.zip
rm $tempdir/couchness.zip
mv couchness* "$install_location"
cd - > /dev/null
rm -rf $tempdir

chmod +x "$install_location"

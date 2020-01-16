#!/bin/bash
set -e

if [ -z "$1" ]; then
  TAG=master
else
  TAG=$1
fi

case $(uname -m | sed 'y/ABCDEFGHIJKLMNOPQRSTUVWXYZ/abcdefghijklmnopqrstuvwxyz/') in
  *64)
    ARCH=amd64
    ;;
  *386*)
    ARCH=386
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

echo "Downloading couchness build for $PLATFORM to ~/bin/couchness"
wget -O $HOME/bin/couchness https://raw.githubusercontent.com/highercomve/couchness/${TAG}/build/${PLATFORM}/couchness

chmod +x $HOME/bin/couchness

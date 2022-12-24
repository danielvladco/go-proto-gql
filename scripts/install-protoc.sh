#!/usr/bin/env bash

PROTOC_VERSION=${PROTOC_VERSION:-"21.12"}

ARCH=$(uname -m)
case $ARCH in
  "arm64")
    ARCH="aarch_64"
    ;;
  "arm32")
    ARCH="aarch_32"
    ;;
  "aarch64")
    ARCH="aarch_64"
    ;;
  "aarch32")
    ARCH="aarch_32"
    ;;
esac

OS=$(uname)
case $OS in
  "Darwin")
    OS="osx"
    ;;
  "Linux")
    OS="linux"
    ;;
esac

ARCHIVE_NAME="protoc-$PROTOC_VERSION-$OS-$ARCH"
curl -LO "https://github.com/protocolbuffers/protobuf/releases/download/v$PROTOC_VERSION/$ARCHIVE_NAME.zip"

cleanup () {
  rm -f "$ARCHIVE_NAME.zip"
  rm -rf $ARCHIVE_NAME
}

trap cleanup 0 1 2 3 9
unzip -o $ARCHIVE_NAME -d $ARCHIVE_NAME
GOPATH="$(go env GOPATH)"
rm -f $GOPATH/bin/protoc
mv $ARCHIVE_NAME/bin/protoc $GOPATH/bin
rm -rf $GOPATH/include
mkdir -p $GOPATH/include
mv $ARCHIVE_NAME/include/* $GOPATH/include/google

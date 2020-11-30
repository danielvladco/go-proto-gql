#!/usr/bin/env bash

PROTOC_VERSION=${PROTOC_VERSION:-"3.13.0"}
OS="linux"
if [[ $(uname) == "Darwin" ]]; then
  OS="osx"
fi

ARCHIVE_NAME="protoc-$PROTOC_VERSION-$OS-$(uname -m)"
curl -LO "https://github.com/protocolbuffers/protobuf/releases/download/v$PROTOC_VERSION/$ARCHIVE_NAME.zip"

cleanup () {
  rm -f "$ARCHIVE_NAME.zip"
  rm -rf $ARCHIVE_NAME
}

trap cleanup EXIT
unzip -o $ARCHIVE_NAME -d $ARCHIVE_NAME
GOPATH="$(go env GOPATH)"
mv $ARCHIVE_NAME/bin/protoc $GOPATH/bin
rm -rf $GOPATH/include
mv $ARCHIVE_NAME/include $GOPATH

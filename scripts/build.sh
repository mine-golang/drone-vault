#!/bin/sh

# disable go modules
export GOPATH=""

# disable cgo
export CGO_ENABLED=0

set -e
set -x

# linux
GOOS=linux GOARCH=amd64 go build -o release/amd64/drone-vault
GOOS=linux GOARCH=arm64 go build -o release/arm64/drone-vault
GOOS=linux GOARCH=arm   go build -o release/arm/drone-vault
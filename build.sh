#!/bin/bash

GOOS=windows GOARCH=amd64 go build -o ./build/socks2tcp.exe
GOOS=windows GOARCH=386 go build -o ./build/socks2tcp_386.exe
GOOS=darwin GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-s -w -extldflags=-static" -o ./build/socks2tcp_amd64_darwin
GOOS=darwin GOARCH=arm64 go build -a -installsuffix cgo -ldflags="-s -w -extldflags=-static" -o ./build/tsnet-exit-node_arm64_darwin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-s -w -extldflags=-static" -o ./build/socks2tcp_amd64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -installsuffix cgo -ldflags="-s -w -extldflags=-static" -o ./build/socks2tcp_arm64
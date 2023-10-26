.PHONY: build
BIN_FILE=idgensvr

build:
	go version
	go build -o "${BIN_FILE}" ./cmd/main.go
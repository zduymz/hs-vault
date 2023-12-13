.PHONY: build test e2e
SHELL := /bin/bash
PROJECT_NAME = hs-vault


build:
	mkdir -p ./dist
	go build -o ./dist/$(PROJECT_NAME) ./cmd

test:
	go test -v ./...

e2e: build
	./e2e/run.sh

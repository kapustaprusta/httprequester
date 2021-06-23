.PHONY: build
build:
	go build -o build/httprequester -v httprequester.go

.PHONY: test
test:
	go test -v -race -timeout 30s ./...

.PHONY: install
install:
	cp build/httprequester /usr/local/bin/

.DEFAULT_GOAL := build
.PHONY: build
build:
	go build -o build/httprequester -v httprequester.go

.PHONY: test
test:
	go test -v -race -timeout 30s ./...

.DEFAULT_GOAL := build
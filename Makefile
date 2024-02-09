.PHONY: build
build:
	go build -o ./bin/nostr-relay-proxy .

.PHONY: run
run:
	go run *.go

.PHONY: test
test:
	go test -v ./...

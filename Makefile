.PHONY: build
build:
	go build -o ./bin/nostr-relay-proxy .

.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 go build -o ./bin/nostr-relay-proxy-linux-amd64 .

.PHONY: run
run:
	go run *.go

.PHONY: test
test:
	go test -v ./...

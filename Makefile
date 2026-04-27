.PHONY: build test clean install

BIN := mkdocs-server

build:
	go build -o $(BIN) ./cmd

test:
	go test ./...

clean:
	rm -f $(BIN) mkdocs-server-*

install: build
	cp $(BIN) /usr/local/bin/$(BIN)

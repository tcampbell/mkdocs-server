.PHONY: build test clean install

BIN := mkdocs-server

build:
	go build -o $(BIN) ./cmd

test:
	go test ./...

clean:
	rm -f $(BIN) mkdocs-server-*

install: build
	install -d ~/.local/bin
	cp $(BIN) ~/.local/bin/$(BIN)

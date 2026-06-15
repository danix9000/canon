BINARY = canon
SYMLINK = $(HOME)/code/danix-scripts/bin/$(BINARY)

.PHONY: build release install test lint precommit

build:
	go build -o $(BINARY) .

release:
	go build -ldflags="-s -w" -trimpath -o $(BINARY) .

install: release
	ln -sf $(CURDIR)/$(BINARY) $(SYMLINK)

test:
	go test -v -count=1 ./...

lint:
	go vet ./...

precommit: build lint test

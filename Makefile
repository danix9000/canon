.PHONY: build test lint precommit

build:
	go build -o canon .

test:
	go test -v -count=1 ./...

lint:
	go vet ./...

precommit: build lint test

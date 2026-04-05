.PHONY: all lint vet test

all: lint vet test

lint:
	.bin/golangci-lint run --fix

vet:
	go vet ./...

fix:
	go fix ./...

test:
	go test -race -count 1 -cover ./...

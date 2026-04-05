.PHONY: all lint lint-fix vet test install-tools go-install-tools

all: lint-fix lint vet test

lint:
	.bin/golangci-lint run

lint-fix:
	.bin/golangci-lint run --fix

vet:
	go vet ./...

fix:
	go fix ./...

test:
	go test -race -count 1 -cover ./...

install-tools: go-install-tools
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b .bin $$(cat .golangci-lint-version)

go-install-tools:
	go install golang.org/x/tools/cmd/goimports@latest

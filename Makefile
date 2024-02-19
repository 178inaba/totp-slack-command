.PHONY: all fmt fmt-diff ci-lint ci-lint-fix vet test install-tools go-install-tools

all: ci-lint-fix fmt ci-lint vet test

fmt:
	goimports -w .

fmt-diff:
	test -z $$(goimports -l .) || (goimports -d . && exit 1)

ci-lint:
	.bin/golangci-lint run

ci-lint-fix:
	.bin/golangci-lint run --fix

vet:
	go vet ./...

test:
	go test -race -count 1 -cover ./...

install-tools: go-install-tools
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b .bin $$(cat .golangci-lint-version)

go-install-tools:
	go install golang.org/x/tools/cmd/goimports@latest

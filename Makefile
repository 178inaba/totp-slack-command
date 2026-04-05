.PHONY: all lint fix vet test

all: lint fix vet test

lint:
	docker compose run --rm lint

fix:
	go fix ./...

vet:
	go vet ./...

test:
	go test -race -count 1 -cover ./...

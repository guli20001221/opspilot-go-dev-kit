GO ?= go

.PHONY: fmt test build check dev-up dev-down

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

build:
	@mkdir -p bin
	$(GO) build -o ./bin/api ./cmd/api
	$(GO) build -o ./bin/worker ./cmd/worker

check: fmt test build

dev-up:
	docker compose up -d

dev-down:
	docker compose down

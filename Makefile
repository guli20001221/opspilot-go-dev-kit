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
	$(GO) build -o ./bin/ticketapi ./cmd/ticketapi

check: fmt test build

dev-up:
	docker compose up -d --build

dev-down:
	docker compose down

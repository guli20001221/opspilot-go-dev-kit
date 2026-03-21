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
	@echo "dev-up is not implemented yet. Next slice should add compose.yaml for Postgres/Redis/Temporal."
	@exit 1

dev-down:
	@echo "dev-down is not implemented yet. No local stack has been defined in this slice."
	@exit 1

GO ?= go

.PHONY: run-api
run-api:
	$(GO) run ./cmd/api

.PHONY: run-matcher
run-matcher:
	$(GO) run ./cmd/matcher

.PHONY: test
test:
	$(GO) test ./...

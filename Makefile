GO ?= go

.PHONY: check
check:
	@test -z "$$(gofmt -l ./*.go)"
	$(GO) vet ./...
	@mkdir -p .artifacts
	$(GO) test -race -count=1 -covermode=atomic -coverpkg=./... -coverprofile=.artifacts/coverage.out ./...
	$(GO) tool cover -func=.artifacts/coverage.out
	@$(GO) tool cover -func=.artifacts/coverage.out | awk '/^total:/ { if ($$3+0 < 97) exit 1 }'
	$(GO) build ./...
	$(GO) mod verify

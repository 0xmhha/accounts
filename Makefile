# stablenet-accounts SDK — developer tasks.
# Go version is declared as 1.23.12 (go-stablenet compatible); a newer local
# toolchain builds it fine. GOTOOLCHAIN=local avoids toolchain downloads.

GO := GOTOOLCHAIN=local go

.PHONY: all test cover vet fmt fmtcheck build e2e live-e2e tidy

all: fmtcheck vet test

## test: run all unit tests (offline, no network)
test:
	$(GO) test ./...

## cover: unit tests with coverage
cover:
	$(GO) test -cover ./...

## vet: go vet
vet:
	$(GO) vet ./...

## fmt: format all Go files in place
fmt:
	gofmt -w .

## fmtcheck: fail if any file is not gofmt-clean
fmtcheck:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "unformatted:"; echo "$$out"; exit 1; fi

## build: compile all packages and commands
build:
	$(GO) build ./...

## e2e: run the SDK e2e against an already-running node (RPC/KEYSTORE overridable)
##   make e2e KEYSTORE=/path/to/keystore RPC=http://127.0.0.1:8505
e2e:
	$(GO) run ./cmd/e2e -rpc $(RPC) -keystore $(KEYSTORE) -password $(PASSWORD)
RPC ?= http://127.0.0.1:8505
PASSWORD ?= 1

## live-e2e: boot a chainbench go-stablenet network, run e2e, tear down
live-e2e:
	./scripts/live-e2e.sh

## tidy: go mod tidy
tidy:
	$(GO) mod tidy

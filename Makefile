GO ?= go
BIN := smaillgeodns
CFG ?= config.yaml

.PHONY: all build run test test-all test-unit test-int test-geo mmdb-clean clean

all: build

build: $(BIN)

$(BIN):
	$(GO) build -o $(BIN) ./cmd/$(BIN)

run: build
	SGDNS_CONFIG=$(CFG) ./$(BIN)

# Tests
test: test-unit test-int

test-all:
	$(GO) test ./...

test-unit:
	$(GO) test ./internal/db ./internal/server/... -count=1

test-int:
	$(GO) test ./internal/integration -count=1

test-geo:
	$(GO) test ./internal/integration -run 'GeoDNS' -count=1

mmdb-clean:
	rm -f ./geoipdb/*.mmdb

clean:
	rm -f $(BIN) *.db *.test *.out smaillgeodns_dev.db

GO ?= go
BIN := namedot
CFG ?= config.yaml

.PHONY: all build run test test-all test-unit test-int test-geo mmdb-clean clean package package-deb package-rpm

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
	rm -f $(BIN) *.db *.test *.out namedot_dev.db *.deb *.rpm

# Package building
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.0.0-dev")

build-for-package:
	@echo "Building namedot binary for packaging (version: $(VERSION))..."
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GO) build -v \
		-ldflags "-X main.Version=$(VERSION) -s -w" \
		-o $(BIN) \
		./cmd/$(BIN)
	@echo "Binary built successfully"
	@ls -lh $(BIN)

package-deb: build-for-package
	@echo "Building DEB package (version: $(VERSION))..."
	VERSION=$(VERSION) nfpm pkg --packager deb --config packaging/nfpm.yaml --target .
	@echo "Package built: $$(ls -1 *.deb | tail -1)"

package-rpm: build-for-package
	@echo "Building RPM package (version: $(VERSION))..."
	VERSION=$(VERSION) nfpm pkg --packager rpm --config packaging/nfpm.yaml --target .
	@echo "Package built: $$(ls -1 *.rpm | tail -1)"

package: package-deb package-rpm
	@echo "All packages built successfully!"
	@ls -lh *.deb *.rpm

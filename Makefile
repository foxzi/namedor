GO ?= go
BIN := smaillgeodns
MMDBGEN := mmdbgen
CFG ?= config.yaml

.PHONY: all build run test test-all test-unit test-int test-geo test-synth mmdbgen mmdb-localhost mmdb mmdb-clean clean

all: build

build: $(BIN)

$(BIN):
	$(GO) build -o $(BIN) ./cmd/$(BIN)

mmdbgen: $(MMDBGEN)

$(MMDBGEN):
	$(GO) build -o $(MMDBGEN) ./cmd/mmdbgen

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

test-synth:
	GEOIP_SYNTH=1 $(GO) test ./internal/integration -run 'SyntheticMMDB' -count=1

# MMDB helpers
mmdb-localhost: mmdbgen
	@mkdir -p geoipdb
	./$(MMDBGEN) -in examples/geoip/localhost.yaml \
		-city-out ./geoipdb/city-localhost.mmdb \
		-asn-out ./geoipdb/asn-localhost.mmdb

# Generic MMDB generation: make mmdb SPEC=examples/geoip/spec.yaml CITY_OUT=./geoipdb/city.mmdb ASN_OUT=./geoipdb/asn.mmdb
mmdb: mmdbgen
	@if [ -z "$(SPEC)" ]; then echo "SPEC=<path to yaml> required"; exit 2; fi
	@mkdir -p $(dir $(CITY_OUT)) $(dir $(ASN_OUT))
	./$(MMDBGEN) -in $(SPEC) -city-out $(CITY_OUT) -asn-out $(ASN_OUT)

mmdb-clean:
	rm -f ./geoipdb/*.mmdb

clean:
	rm -f $(BIN) $(MMDBGEN) *.db *.test *.out smaillgeodns_dev.db


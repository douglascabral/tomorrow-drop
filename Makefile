# Tomorrow Drop build and check targets.
# Compatible with GNU Make 3.81 (the version shipped with macOS).
#
#   make help        list the targets
#   make check       run the full validation required by AGENTS.md

GO        ?= go
FUZZTIME  ?= 15s
BIN       := bin
DIST      := dist
CGO_OFF   := CGO_ENABLED=0
DEV_HOME  ?= /tmp/tomorrow-drop-dev

FUZZ_TARGETS := \
	./internal/protocol:FuzzReadFrame \
	./internal/protocol:FuzzDecodeMessage \
	./internal/discovery:FuzzDecodeDatagram \
	./internal/discovery:FuzzParseEndpoint \
	./internal/transfer:FuzzValidateName \
	./internal/transfer:FuzzValidateLink \
	./internal/identity:FuzzDecodeIdentity \
	./internal/trust:FuzzTrustFile

.DEFAULT_GOAL := help
.PHONY: help build cli gui cross test race vet fmt fmt-fix lint check fuzz bench deps package-macos install-android install-ios run run-cli run-dev clean

help: ## Show this help
	@echo "Tomorrow Drop make targets:"
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | sed -e 's/:.*## /|/' | awk -F'|' '{printf "  %-16s %s\n", $$1, $$2}'

build: cli gui ## Build the CLI and the GUI for this machine

cli: ## Build bin/tdrop (no cgo)
	@mkdir -p $(BIN)
	$(CGO_OFF) $(GO) build -trimpath -o $(BIN)/tdrop ./cmd/tdrop

gui: ## Build bin/tomorrow-drop (needs cgo and the graphics libraries)
	@mkdir -p $(BIN)
	$(GO) build -trimpath -o $(BIN)/tomorrow-drop ./cmd/tomorrow-drop

cross: ## Cross-compile the CLI for Linux and Windows (amd64)
	@mkdir -p $(BIN)
	GOOS=linux   GOARCH=amd64 $(CGO_OFF) $(GO) build -trimpath -o $(BIN)/tdrop-linux-amd64 ./cmd/tdrop
	GOOS=windows GOARCH=amd64 $(CGO_OFF) $(GO) build -trimpath -o $(BIN)/tdrop-windows-amd64.exe ./cmd/tdrop

test: ## Run the test suite
	$(GO) test ./...

race: ## Run the test suite with the race detector
	$(GO) test -race ./...

vet: ## Run go vet
	$(GO) vet ./...

fmt: ## List files that are not gofmt-formatted (fails if any)
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "$$out"; exit 1; fi

fmt-fix: ## Format all Go files in place
	gofmt -w .

lint: ## Run golangci-lint
	golangci-lint run ./...

check: ## Run the full validation (scripts/check.sh)
	scripts/check.sh

fuzz: ## Run every fuzz target for FUZZTIME (default 15s each)
	@for t in $(FUZZ_TARGETS); do \
		pkg=$${t%%:*}; fn=$${t##*:}; \
		echo "== $$fn ($$pkg)"; \
		$(GO) test -run='^$$' -fuzz="^$$fn\$$" -fuzztime=$(FUZZTIME) $$pkg || exit 1; \
	done

bench: ## Run the 64 MiB transfer benchmark once
	$(GO) test -run='^$$' -bench=BenchmarkTransferLargeFile -benchtime=3x ./internal/transfer/

deps: ## Print the license of every module in the build graph
	scripts/dependency-report.sh

package-macos: ## Build the unsigned macOS application bundle into dist/
	scripts/package-macos.sh

install-android: ## Build the Android APK and install it on the connected phone
	scripts/install-android.sh

install-ios: ## Build the iPhone/iPad app and install it on the connected device (scripts/install-ios.sh)
	scripts/install-ios.sh

run: gui ## Build and open the graphical app with your real settings
	$(BIN)/tomorrow-drop

run-dev: gui ## Open the app with a separate home (no identity or settings touched)
	@mkdir -p $(DEV_HOME)
	HOME=$(DEV_HOME) $(BIN)/tomorrow-drop

run-cli: cli ## Run the CLI, e.g. make run-cli ARGS="identity show"
	$(BIN)/tdrop $(ARGS)

clean: ## Remove build outputs
	rm -rf $(BIN) $(DIST)

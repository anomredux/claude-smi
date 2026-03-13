BINARY      := claude-smi
CMD         := ./cmd/claude-smi
GO          ?= go
GOBIN       ?= $(shell $(GO) env GOPATH)/bin
COVERAGE    := coverage.out
COVER_PKGS  := $(shell $(GO) list ./... | grep -v -E '/(ui|theme)')

.PHONY: all build test test-short test-race cover cover-html fmt fix vet lint check clean install uninstall

## all: fmt + vet + test-short + lint + build (default target)
all: fmt vet test-short lint build

## build: compile the binary
build:
	$(GO) build -o $(BINARY) $(CMD)

## fmt: verify code formatting
fmt:
	@echo "Checking formatting..."
	@test -z "$$(gofmt -l .)" || (echo "Unformatted files:"; gofmt -l .; exit 1)

## fix: run go fix to modernize code
fix:
	@echo "Running go fix..."
	$(GO) fix ./...

## test: run all tests
test:
	$(GO) test -count=1 ./...

## test-short: run tests in short mode
test-short:
	$(GO) test -short -count=1 ./...

## test-race: run all tests with race detector
test-race:
	$(GO) test -race -count=1 ./...

## cover: run tests with coverage report (excludes ui/theme packages)
cover:
	$(GO) test -race -coverprofile=$(COVERAGE) -covermode=atomic $(COVER_PKGS)
	$(GO) tool cover -func=$(COVERAGE)

## cover-html: open coverage report in browser
cover-html: cover
	$(GO) tool cover -html=$(COVERAGE)

## vet: run go vet
vet:
	$(GO) vet ./...

## lint: golangci-lint (installs if missing)
lint:
	@test -x $(GOBIN)/golangci-lint || $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOBIN)/golangci-lint run

## check: test-race + lint
check: test-race lint

## clean: remove build artifacts
clean:
	rm -f $(BINARY) $(BINARY).exe $(COVERAGE)

## install: install binary to GOPATH/bin
install:
	$(GO) install $(CMD)

## uninstall: remove binary from GOPATH/bin
uninstall:
	rm -f $(GOBIN)/$(BINARY)

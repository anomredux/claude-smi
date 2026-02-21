BINARY      := claude-smi
CMD         := ./cmd/claude-smi
GOBIN       ?= $(shell go env GOPATH)/bin
COVERAGE    := coverage.out
COVER_PKGS  := $(shell go list ./... | grep -v -E '/(ui|theme)')

.PHONY: all build test test-race cover cover-html vet lint staticcheck check clean install uninstall

## all: build + check + cover (default target)
all: build check cover

## build: compile the binary
build:
	go build -o $(BINARY) $(CMD)

## test: run all tests
test:
	go test ./... -count=1

## test-race: run all tests with race detector
test-race:
	go test -race ./... -count=1

## cover: run tests with coverage report (excludes ui/theme packages)
cover:
	go test -race -coverprofile=$(COVERAGE) -covermode=atomic $(COVER_PKGS)
	go tool cover -func=$(COVERAGE)

## cover-html: open coverage report in browser
cover-html: cover
	go tool cover -html=$(COVERAGE)

## vet: run go vet
vet:
	go vet ./...

## staticcheck: run staticcheck (installs if missing)
staticcheck:
	@test -x $(GOBIN)/staticcheck || go install honnef.co/go/tools/cmd/staticcheck@latest
	$(GOBIN)/staticcheck ./...

## lint: vet + staticcheck
lint: vet staticcheck

## check: test-race + lint
check: test-race lint

## clean: remove build artifacts
clean:
	rm -f $(BINARY) $(COVERAGE)

## install: install binary to GOPATH/bin
install:
	go install $(CMD)

## uninstall: remove binary from GOPATH/bin
uninstall:
	rm -f $(GOBIN)/$(BINARY)

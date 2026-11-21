# Single entry point for the project's checks. CI runs `make check` too.
#
# Must work with GNU Make 3.81 (macOS /usr/bin/make): no .ONESHELL, no `!=`,
# no $(file ...), no grouped targets, no `::=`. Recipes are plain POSIX sh so
# they also run under Git Bash on Windows.

# The only place the golangci-lint version is written. CI reads it through
# the golangci-lint-version target.
GOLANGCI_LINT_VERSION := v2.15.0

# Platforms where `go test -race` is supported (see `go help build`).
RACE_PLATFORMS := darwin/amd64 darwin/arm64 freebsd/amd64 linux/amd64 linux/arm64 linux/ppc64le linux/riscv64 windows/amd64

VERSION ?= dev

ifeq ($(OS),Windows_NT)
# Use Git Bash's sh from PATH instead of falling back to cmd.exe.
SHELL := sh
endif

BIN := mhz$(shell go env GOEXE)

.PHONY: check fmt-check tidy-check vet lint test race build golangci-lint-version
.NOTPARALLEL:

check: fmt-check tidy-check vet lint test race

fmt-check:
	@echo "==> gofmt -l ."
	@out=$$(gofmt -l .) || exit 1; \
	if [ -n "$$out" ]; then \
		echo "gofmt would change these files:"; \
		echo "$$out"; \
		exit 1; \
	fi

tidy-check:
	@echo "==> go mod tidy -diff"
	@go mod tidy -diff

vet:
	@echo "==> go vet ./..."
	@go vet ./...

lint:
	@echo "==> golangci-lint run"
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found on PATH; install $(GOLANGCI_LINT_VERSION) with:" >&2; \
		echo "  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)" >&2; \
		exit 1; \
	}
	@golangci-lint run

test:
	@echo "==> go test ./..."
	@go test ./...

race:
	@echo "==> go test -race ./..."
	@goos=$$(go env GOOS); goarch=$$(go env GOARCH); cgo=$$(go env CGO_ENABLED); \
	case " $(RACE_PLATFORMS) " in \
		*" $$goos/$$goarch "*) ;; \
		*) echo "skipping race: unsupported platform $$goos/$$goarch"; exit 0 ;; \
	esac; \
	if [ "$$cgo" != "1" ]; then \
		echo "skipping race: cgo is disabled"; \
		exit 0; \
	fi; \
	go test -race ./...

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BIN) .

golangci-lint-version:
	@echo $(GOLANGCI_LINT_VERSION)

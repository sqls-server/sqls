NAME := sqls
VERSION := v0.2.1
REVISION := $(shell git rev-parse --short HEAD)
GOVERSION := $(go version)

SRCS := $(shell find . -type f -name '*.go')
LDFLAGS := -ldflags="-s -w -X \"main.version=$(VERSION)\" -X \"main.revision=$(REVISION)\""
DIST_DIRS := find * -type d -exec

.PHONY: test
test:
	go test ./...

.PHONY: build
build: $(SRCS)
	go build $(LDFLAGS) ./...

.PHONY: install
install: $(SRCS)
	go install $(LDFLAGS) ./...

.PHONY: lint
lint: $(SRCS)
	golangci-lint run

.PHONY: coverage
coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: stringer
stringer:
	stringer -type Kind ./token/kind.go

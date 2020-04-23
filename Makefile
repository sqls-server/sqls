NAME := sqls
VERSION := v0.1.0
REVISION := $(shell git rev-parse --short HEAD)
GOVERSION := $(go version)
SRCS := $(shell find . -type f -name '*.go')
LDFLAGS := -ldflags="-s -w -X \"main.version=$(VERSION)\" -X \"main.revision=$(REVISION)\""

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

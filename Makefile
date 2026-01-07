BIN := sqls
ifeq ($(OS),Windows_NT)
BIN := $(BIN).exe
endif
VERSION = $$(make -s show-version)
CURRENT_REVISION := $(shell git rev-parse --short HEAD)
BUILD_LDFLAGS := "-s -w -X main.revision=$(CURRENT_REVISION)"
GOOS := $(shell go env GOOS)
GOBIN ?= $(shell go env GOPATH)/bin
export GO111MODULE=on

.PHONY: all
all: clean build

.PHONY: build
build:
	go build -ldflags=$(BUILD_LDFLAGS) -o $(BIN) .

.PHONY: release
release: $(GOBIN)/gobump
	go build -ldflags=$(BUILD_LDFLAGS) -o $(BIN) .
	zip -r sqls-$(GOOS)-$(VERSION).zip $(BIN)

.PHONY: install
install:
	go install -ldflags=$(BUILD_LDFLAGS) .

.PHONY: show-version
show-version: $(GOBIN)/gobump
	@$(GOBIN)/gobump show -r .

$(GOBIN)/gobump:
	env GOOS=$(shell go env GOHOSTOS) GOARCH=$(shell go env GOHOSTARCH) go install github.com/x-motemen/gobump/cmd/gobump@latest

.PHONY: test
test: build
	go test -v ./...

.PHONY: clean
clean:
	go clean

.PHONY: bump
bump: $(GOBIN)/gobump
ifneq ($(shell git status --porcelain),)
	$(error git workspace is dirty)
endif
ifneq ($(shell git rev-parse --abbrev-ref HEAD),master)
	$(error current branch is not master)
endif
	@$(GOBIN)/gobump up -w .
	git commit -am "bump up version to $(VERSION)"
	git tag "v$(VERSION)"
	git push origin master
	git push origin "refs/tags/v$(VERSION)"

NAME := sqls
VERSION := $(shell git describe --tags `git rev-list --tags --max-count=1`)
REVISION := $(shell git rev-parse --short HEAD)
GOVERSION := $(go version)
GITHUB_TOKEN := $(GITHUB_TOKEN)

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

.PHONY: snapshot
snapshot: $(SRCS)
	docker run --rm --privileged \
		-v ${PWD}:/go/src/github.com/lighttiger2505/sqls \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /go/src/github.com/lighttiger2505/sqls \
		mailchain/goreleaser-xcgo --snapshot --rm-dist

.PHONY: publish
publish: $(SRCS)
	docker run --rm --privileged \
		-e GITHUB_TOKEN=$(GITHUB_TOKEN) \
		-v ${PWD}:/go/src/github.com/lighttiger2505/sqls \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /go/src/github.com/lighttiger2505/sqls \
		mailchain/goreleaser-xcgo --rm-dist

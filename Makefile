NAME := sqls
VERSION := v0.1.0
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

.PHONY: cross-build
cross-build:
	for os in darwin linux windows; do \
		for arch in amd64 386; do \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$$os-$$arch/$(NAME); \
		done; \
	done

.PHONY: dist
dist: cross-build
	cd dist && \
	$(DIST_DIRS) cp ../LICENSE {} \; && \
	$(DIST_DIRS) cp ../README.md {} \; && \
	$(DIST_DIRS) tar -zcf $(NAME)-$(VERSION)-{}.tar.gz {} \; && \
	$(DIST_DIRS) zip -r $(NAME)-$(VERSION)-{}.zip {} \; && \
	cd ..

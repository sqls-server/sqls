SRCS := $(shell find . -type f -name '*.go')
LDFLAGS := -ldflags="-s -w -X \"main.version=$(VERSION)\" -X \"main.revision=$(REVISION)\" -X \"main.goversion=$(GOVERSION)\" "

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

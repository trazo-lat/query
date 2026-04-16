.PHONY: build test coverage lint fmt vet check clean wasm

GO       := go
GOTEST   := $(GO) test
GOFLAGS  := -race
COVERAGE := coverage.out

build:
	$(GO) build ./...

test:
	$(GOTEST) $(GOFLAGS) -count=1 ./...

coverage:
	$(GOTEST) $(GOFLAGS) -coverprofile=$(COVERAGE) -covermode=atomic ./...
	$(GO) tool cover -func=$(COVERAGE) | tail -1

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w -local github.com/trazo-lat/query .

vet:
	$(GO) vet ./...

check: fmt vet lint test

clean:
	rm -rf bin/ coverage.out coverage.html wasm/*.wasm

wasm:
	cd wasm && $(MAKE) build

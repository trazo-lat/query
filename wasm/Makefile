.PHONY: build clean setup

GOROOT := $(shell go env GOROOT)

build:
	GOOS=js GOARCH=wasm go build -o query.wasm .
	cp "$(GOROOT)/lib/wasm/wasm_exec.js" npm/src/wasm_exec.js

setup: build
	cd npm && npm install

clean:
	rm -f query.wasm npm/src/wasm_exec.js
	rm -rf npm/dist npm/node_modules

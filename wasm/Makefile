.PHONY: build clean

build:
	GOOS=js GOARCH=wasm go build -o query.wasm .

clean:
	rm -f query.wasm

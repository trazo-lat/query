//go:build wasm

// Package main is the WASM entry point for the query library.
// Phase 4: will expose parse() and validate() to JavaScript via syscall/js.
package main

func main() {
	// TODO(trazo): Phase 4 — register JS functions for parse and validate.
	// This is a scaffold for the WASM build target.
}

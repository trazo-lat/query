// End-to-end tests for the query WASM bridge, exercised through the Go binary
// from JavaScript. Run with the project's zero-dependency Node test runner:
//
//   make -C wasm build   # produces clients/npm/query.wasm + src/wasm_exec.js
//   node --test
//
// These prove the JS <-> WASM round-trip for every exported function, including
// the match/eval bridges, not just the Go-side unit tests.

import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const here = dirname(fileURLToPath(import.meta.url));
const wasmPath = join(here, "..", "query.wasm");
const wasmExecPath = join(here, "..", "src", "wasm_exec.js");

// Load Go's wasm_exec.js shim (defines globalThis.Go) and instantiate the
// module once for the whole suite.
async function loadWasm() {
  await import(wasmExecPath);
  const go = new globalThis.Go();
  const bytes = readFileSync(wasmPath);
  const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
  void go.run(instance); // main uses select{}; never resolves — do not await
  return globalThis;
}

const fields = [
  { Name: "state", Type: 0, AllowedOps: ["=", "!=", "*", "?"] }, // TypeText
  { Name: "total", Type: 2, AllowedOps: ["=", "!=", ">", ">=", "<", "<=", ".."] }, // TypeDecimal
  { Name: "base", Type: 2, AllowedOps: ["=", "!=", ">", ">=", "<", "<=", ".."] },
];

const w = await loadWasm();

test("queryParse returns an AST", () => {
  const { result, error } = w.queryParse("state=draft AND total>50000");
  assert.equal(error, undefined);
  assert.equal(result.type, "binary");
});

test("queryParse reports a parse error", () => {
  const { error } = w.queryParse("=invalid");
  assert.ok(error, "expected a parse error");
});

// A client renders its own copy of a failure — a Spanish UI underlining the
// token that broke — so the joined English sentence is not enough. These pin
// the structured half of the contract.
test("queryParse reports each failure with a code and a span", () => {
  const { errors } = w.queryParse("items@nope(x=1)");
  assert.ok(Array.isArray(errors), "expected a structured errors array");
  assert.ok(errors.length > 0, "expected at least one structured error");

  const [first] = errors;
  assert.equal(first.code, "expectedSelector");
  assert.equal(first.offset, 6, "offset must point at the offending token");
  assert.equal(first.length, 4, "length must span it, so a caret can underline");
  assert.equal(typeof first.kind, "string");
  assert.ok(first.message.length > 0);
});

test("queryParse codes the common failures a person types", () => {
  const cases = [
    ["(state=draft", "unclosedParen"],
    ["state=", "expectedValue"],
    ["=draft", "expectedField"],
    ["state IN ()", "emptyInList"],
    ["state=**b**", "invalidWildcard"],
    ['state="draft', "unclosedString"],
  ];
  for (const [query, code] of cases) {
    const { errors } = w.queryParse(query);
    assert.ok(errors?.length, `${query}: expected structured errors`);
    assert.equal(errors[0].code, code, `${query}: wrong code`);
  }
});

test("queryParse carries a zero-length span at end of input", () => {
  // A failure at the end has nowhere to underline. Zero is the honest answer
  // and must survive the wire rather than being dropped as falsy.
  const { errors } = w.queryParse("state=");
  assert.equal(errors[0].length, 0);
  assert.ok(Object.hasOwn(errors[0], "length"), "length must be present, not omitted");
});

test("queryParse omits the structured array when it succeeds", () => {
  const { result, errors, error } = w.queryParse("state=draft");
  assert.ok(result, "expected an AST");
  assert.equal(error, undefined);
  assert.equal(errors, undefined, "a success carries no errors array");
});

test("queryParseAndValidate reports parse failures structurally too", () => {
  const { errors } = w.queryParseAndValidate("(state=draft", JSON.stringify(fields));
  assert.ok(errors?.length, "expected structured errors from the validate entry point");
  assert.equal(errors[0].code, "unclosedParen");
});

test("a validation failure carries no parse errors array", () => {
  // Only a PARSE failure is coded this way; a validation failure is a
  // different envelope and must not pretend otherwise.
  const { errors, error } = w.queryParseAndValidate("color=red", JSON.stringify(fields));
  assert.ok(error, "expected a validation failure");
  assert.equal(errors, undefined);
});

test("queryParseAndValidate accepts a valid query", () => {
  const { result, error } = w.queryParseAndValidate(
    "state=draft",
    JSON.stringify(fields)
  );
  assert.equal(error, undefined);
  assert.equal(result.type, "qualifier");
});

test("queryValidate flags an unknown field", () => {
  const { result: ast } = w.queryParse("state=draft");
  const v = w.queryValidate(JSON.stringify(ast), JSON.stringify([]));
  assert.equal(v.valid, false);
  assert.ok(v.errors.length > 0);
});

test("queryStringify round-trips a query", () => {
  const { result: ast } = w.queryParse("state=draft AND total>50000");
  const { result: str } = w.queryStringify(JSON.stringify(ast));
  assert.equal(str, "state=draft AND total>50000");
});

test("queryMatch evaluates a boolean predicate", () => {
  const f = JSON.stringify(fields);
  const yes = w.queryMatch(
    "state=draft AND total>50000",
    f,
    JSON.stringify({ state: "draft", total: 60000 })
  );
  assert.equal(yes.error, undefined);
  assert.equal(yes.result, true);

  const no = w.queryMatch(
    "total>100000",
    f,
    JSON.stringify({ state: "draft", total: 60000 })
  );
  assert.equal(no.result, false);
});

test("queryMatch supports cross-field comparison", () => {
  const f = JSON.stringify(fields);
  const r = w.queryMatch(
    "total>[base]",
    f,
    JSON.stringify({ total: 100, base: 50 })
  );
  assert.equal(r.error, undefined);
  assert.equal(r.result, true);
});

test("queryEval computes a value expression", () => {
  const r = w.queryEval(
    "[base]*2",
    JSON.stringify(fields),
    JSON.stringify({ base: 21 })
  );
  assert.equal(r.error, undefined);
  assert.equal(r.result, 42);
});

test("queryEval reports ErrNoValue for a missing field", () => {
  const r = w.queryEval(
    "[base]",
    JSON.stringify(fields),
    JSON.stringify({})
  );
  assert.ok(r.error, "expected an error for a missing field");
});

test("queryMatch surfaces a compile error", () => {
  const r = w.queryMatch(
    "unknown_field=x",
    JSON.stringify(fields),
    JSON.stringify({})
  );
  assert.ok(r.error, "expected a validate/compile error");
});

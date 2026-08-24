import assert from "node:assert/strict";
import { test } from "node:test";
import { greet } from "./index.ts";

test("greet names its argument", () => {
  assert.equal(greet("seed"), "hello, seed");
});

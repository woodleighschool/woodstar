import assert from "node:assert/strict";
import { test } from "node:test";

import { idParams } from "./route-params.ts";

void test("a path id is a canonical positive integer", () => {
  assert.deepEqual(idParams.parse({ id: "1" }), { id: 1 });
  assert.deepEqual(idParams.parse({ id: "9007199254740991" }), { id: 9007199254740991 });
  for (const id of ["", "0", "-1", "1.5", "1e2", "01", " 1", "0x10", "abc", "9007199254740992"]) {
    assert.equal(idParams.parse({ id }), false, `"${id}" is not an id`);
  }
});

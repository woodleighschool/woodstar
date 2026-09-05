import assert from "node:assert/strict";
import { test } from "node:test";

import type { NavItem } from "./nav-config";
import { filterNavigation, firstNavigationTarget } from "./navigation.ts";

void test("navigation sends a restricted group to its first accessible descendant", () => {
  const items: NavItem[] = [
    {
      label: "Munki",
      to: "/munki",
      items: [
        {
          label: "Software",
          to: "/munki/software",
          permission: { resource: "munki.software", access: "view" },
        },
        {
          label: "Packages",
          to: "/munki/packages",
          permission: { resource: "munki.packages", access: "view" },
        },
      ],
    },
    { label: "Disabled", to: "/disabled", disabled: true },
  ];
  const original = structuredClone(items);
  const filtered = filterNavigation(items, { "munki.packages": "view" });
  assert.equal(filtered[0]?.to, "/munki/packages");
  assert.equal(filtered[0]?.items?.length, 1);
  assert.equal(firstNavigationTarget(filtered), "/munki/packages");
  assert.deepEqual(filterNavigation(items, undefined), []);
  assert.deepEqual(items, original);
});

import js from "@eslint/js";
import stylistic from "@stylistic/eslint-plugin";
import prettier from "eslint-config-prettier";
import reactDom from "eslint-plugin-react-dom";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import reactX from "eslint-plugin-react-x";
import unusedImports from "eslint-plugin-unused-imports";
import { globalIgnores } from "eslint/config";
import globals from "globals";
import tseslint from "typescript-eslint";

export default tseslint.config([
  globalIgnores([
    "dist",
    "pnpm-lock.yaml",
    "vite.config.ts",
    "src/components/ui/**",
    "src/routeTree.gen.ts",
    "src/lib/api-client/**",
  ]),
  {
    files: ["**/*.{ts,tsx}"],
    extends: [js.configs.recommended, tseslint.configs.recommendedTypeChecked, reactRefresh.configs.vite, prettier],
    languageOptions: {
      ecmaVersion: 2022,
      globals: globals.browser,
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      "react-hooks": reactHooks,
      "react-x": reactX,
      "react-dom": reactDom,
      "unused-imports": unusedImports,
      "@stylistic": stylistic,
    },
    rules: {
      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "warn",
      "react-refresh/only-export-components": ["warn", { allowConstantExport: true }],

      "react-x/no-array-index-key": "warn",
      "react-x/no-leaked-conditional-rendering": "error",
      "react-x/no-duplicate-key": "error",
      "react-dom/no-dangerously-set-innerhtml": "warn",
      "react-dom/no-missing-button-type": "warn",
      "react-dom/no-unsafe-target-blank": "error",

      "unused-imports/no-unused-imports": "error",
      "@typescript-eslint/no-unused-vars": "off",
      "unused-imports/no-unused-vars": [
        "warn",
        { vars: "all", varsIgnorePattern: "^_", args: "after-used", argsIgnorePattern: "^_" },
      ],

      "@typescript-eslint/no-deprecated": "warn",
      "@typescript-eslint/consistent-type-imports": ["warn", { fixStyle: "separate-type-imports" }],
      "@typescript-eslint/no-explicit-any": "warn",
      "@typescript-eslint/no-unnecessary-condition": "warn",
      "@typescript-eslint/prefer-nullish-coalescing": "warn",
      "@typescript-eslint/prefer-optional-chain": "warn",

      "@typescript-eslint/no-misused-promises": "warn",
      "@typescript-eslint/no-unnecessary-type-assertion": "warn",
      "@typescript-eslint/no-base-to-string": "warn",
      "@typescript-eslint/restrict-template-expressions": ["warn", { allowNumber: true, allowBoolean: true }],

      "@stylistic/quotes": ["warn", "double", { avoidEscape: true }],
      "no-console": ["warn", { allow: ["warn", "error"] }],
    },
  },
  {
    // fields.tsx deliberately co-locates a resource's form component with its
    // schema/defaults/mappers, so the component-only fast-refresh rule does not apply.
    files: ["src/pages/**/fields.tsx"],
    rules: { "react-refresh/only-export-components": "off" },
  },
  {
    // Vendored diceui registry code (data-table primitives + their support
    // hooks/utils/types). Kept close to upstream, so it is exempt from the
    // app-strict rules it trips, same treatment as the shadcn ui/** primitives.
    // Our own additions to the data-table dir stay under the strict rules.
    files: [
      "src/components/data-table/**/*.{ts,tsx}",
      "src/hooks/use-data-table.ts",
      "src/hooks/use-debounced-callback.ts",
      "src/hooks/use-callback-ref.ts",
      "src/lib/data-table.ts",
      "src/lib/parsers.ts",
      "src/lib/compose-refs.ts",
      "src/lib/format.ts",
      "src/types/data-table.ts",
      "src/config/data-table.ts",
    ],
    ignores: ["src/components/data-table/data-table-static.tsx"],
    rules: {
      "@typescript-eslint/no-unnecessary-condition": "off",
      "@typescript-eslint/no-unsafe-assignment": "off",
      "@typescript-eslint/no-floating-promises": "off",
      "@typescript-eslint/prefer-nullish-coalescing": "off",
      "react-x/no-leaked-conditional-rendering": "off",
      "react-x/no-array-index-key": "off",
      "react-hooks/exhaustive-deps": "off",
      "unused-imports/no-unused-vars": "off",
    },
  },
]);

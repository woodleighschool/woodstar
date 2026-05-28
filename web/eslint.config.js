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
]);

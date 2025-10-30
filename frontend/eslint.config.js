import js from "@eslint/js";
import svelte from "eslint-plugin-svelte";
import globals from "globals";
import ts from "typescript-eslint";

/** @type {import('eslint').Linter.Config[]} */
export default [
  js.configs.recommended,
  ...ts.configs.recommended,
  ...svelte.configs["flat/recommended"],
  {
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node
      }
    }
  },
  {
    files: ["**/*.svelte"],

    languageOptions: {
      parserOptions: {
        parser: ts.parser
      }
    },
    rules: {
      // High-churn Svelte authoring: keep these as warnings to avoid blocking CI
      "@typescript-eslint/no-explicit-any": "warn",
      "no-empty": "warn"
    }
  },
  {
    ignores: ["build/", ".svelte-kit/", "dist/"]
  }
];

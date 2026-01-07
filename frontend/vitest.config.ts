import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "jsdom",
    coverage: {
      provider: "v8",
      include: ["src/v2/**/*.ts"],
      exclude: ["**/*.bench.ts"],
    },
    silent: "passed-only",
  },
});

import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  timeout: 60_000,
  use: { viewport: { width: 1440, height: 900 } },
  reporter: "list",
});

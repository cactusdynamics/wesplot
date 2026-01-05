import { resolve } from "node:path";
import { visualizer } from "rollup-plugin-visualizer";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [
    // This will output size visualization for the JS bundle at stats.html in
    // this folder.
    visualizer(),
  ],
  build: {
    rollupOptions: {
      input: {
        main: resolve(__dirname, "index.html"),
        v2: resolve(__dirname, "v2.html"),
      },
    },
  },
});

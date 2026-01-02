import { visualizer } from "rollup-plugin-visualizer";
import { resolve } from "path";

export default {
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
};

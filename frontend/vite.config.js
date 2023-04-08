import { visualizer } from "rollup-plugin-visualizer";

export default {
  plugins: [
    // This will output size visualization for the JS bundle at stats.html in
    // this folder.
    visualizer(),
  ],
};

/**
 * Benchmarks for Streamer component
 *
 * This will benchmark protocol decoding, data buffering,
 * and callback dispatch performance.
 */

import { bench, describe } from "vitest";

describe("Streamer performance", () => {
  bench("protocol decoding", () => {
    // Placeholder: Benchmark binary protocol decoding
    const data = new Float64Array(1000);
    for (let i = 0; i < data.length; i++) {
      data[i] = Math.random();
    }
  });

  bench("data buffering", () => {
    // Placeholder: Benchmark data append to circular buffer
    const buffer = new Float64Array(10000);
    for (let i = 0; i < 1000; i++) {
      buffer[i % buffer.length] = i;
    }
  });

  bench("callback dispatch", () => {
    // Placeholder: Benchmark callback invocation overhead
    const callback = (
      _seriesId: number,
      _x: Float64Array[],
      _y: Float64Array[],
    ) => {
      // No-op callback
    };
    const x = [new Float64Array(100)];
    const y = [new Float64Array(100)];
    callback(0, x, y);
  });
});

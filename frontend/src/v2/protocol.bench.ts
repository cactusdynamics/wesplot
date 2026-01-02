/**
 * Benchmarks for binary protocol decoder
 *
 * This will benchmark parsing speed for different message types.
 */

import { bench, describe } from "vitest";

describe("Protocol performance", () => {
  bench("envelope header parsing", () => {
    // Placeholder: Benchmark DataView operations for header
    const buffer = new ArrayBuffer(8);
    const view = new DataView(buffer);
    view.setUint8(0, 1); // version
    view.setUint8(1, 0); // reserved
    view.setUint16(2, 1, false); // type (big-endian)
    view.setUint32(4, 1024, false); // length (big-endian)

    const version = view.getUint8(0);
    const type = view.getUint16(2, false);
    const length = view.getUint32(4, false);
    version + type + length; // Use the values
  });

  bench("DATA message decode", () => {
    // Placeholder: Benchmark extracting seriesId and Float64Arrays
    const buffer = new ArrayBuffer(16 + 8 * 200); // seriesId + length + 100 points
    const view = new DataView(buffer);
    view.setUint32(0, 0, false); // seriesId
    view.setUint32(4, 100, false); // length

    const seriesId = view.getUint32(0, false);
    const length = view.getUint32(8, false);
    const xArray = new Float64Array(buffer, 16, length);
    const yArray = new Float64Array(buffer, 16 + length * 8, length);

    seriesId + xArray.length + yArray.length; // Use the values
  });

  bench("JSON metadata parse", () => {
    // Placeholder: Benchmark JSON.parse for metadata
    const json = '{"series":[{"id":0,"name":"series0"}],"options":{}}';
    const metadata = JSON.parse(json);
    metadata.series.length; // Use the result
  });
});

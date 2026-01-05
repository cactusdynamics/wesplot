import { bench, describe } from "vitest";
import { CircularBuffer } from "./circular_buffer.js";

const N = 100_000;
const CAP = 1024;

describe("CircularBuffer performance", () => {
  bench("CircularBuffer append N items", () => {
    const b = new CircularBuffer(CAP);
    for (let i = 0; i < N; i++) {
      b.append(i);
    }
    const s = b.segments();
    if (s.length === 0) throw new Error("no segments");
  });

  bench("Naive Array push/shift N items", () => {
    const a: number[] = [];
    for (let i = 0; i < N; i++) {
      a.push(i);
      if (a.length > CAP) a.shift();
    }
    if (a.length === 0) throw new Error("no items");
  });
});

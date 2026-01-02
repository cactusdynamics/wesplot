import { describe, expect, it } from "vitest";
import { CircularBuffer } from "./circular_buffer.js";

function concatSegments(segs: Float64Array[]): number[] {
  const out: number[] = [];
  for (const s of segs) {
    for (let i = 0; i < s.length; i++) out.push(s[i]);
  }
  return out;
}

describe("CircularBuffer", () => {
  it("returns empty segments for new buffer", () => {
    const b = new CircularBuffer(4);

    expect(b.length()).toBe(0);
    expect(b.segments()).toEqual([]);
  });

  it("append exactly capacity produces single contiguous segment", () => {
    const b = new CircularBuffer(4);
    b.append(1, 2, 3, 4);

    expect(b.length()).toBe(4);
    const segs = b.segments();
    expect(segs.length).toBe(1);
    expect(concatSegments(segs)).toEqual([1, 2, 3, 4]);
  });

  it("overflow drops oldest values and preserves order", () => {
    const b = new CircularBuffer(3);
    b.append(1, 2, 3, 4);

    expect(b.length()).toBe(3);
    expect(concatSegments(b.segments())).toEqual([2, 3, 4]);
  });

  it("multiple wrap-arounds behave correctly", () => {
    const b = new CircularBuffer(4);
    // append 10 numbers, expect last 4
    const nums = Array.from({ length: 10 }, (_, i) => i + 1);
    b.append(...nums);

    expect(b.length()).toBe(4);
    expect(concatSegments(b.segments())).toEqual([7, 8, 9, 10]);
  });

  it("appendNaN inserts NaN sentinel preserved in segments", () => {
    const b = new CircularBuffer(4);
    b.append(1, 2);
    b.append(Number.NaN);
    b.append(3);

    const out = concatSegments(b.segments());
    expect(out.length).toBe(4);
    expect(out[0]).toBe(1);
    expect(Number.isNaN(out[2])).toBe(true);
    expect(out[1]).toBe(2);
    expect(out[3]).toBe(3);
  });
});

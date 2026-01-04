export class CircularBuffer {
  private buf: Float64Array;
  private start: number; // index of oldest element
  private count: number; // number of elements stored

  constructor(private cap: number) {
    if (!Number.isInteger(cap) || cap <= 0) {
      throw new Error("capacity must be a positive integer");
    }
    this.buf = new Float64Array(cap);
    this.start = 0;
    this.count = 0;
  }

  capacity(): number {
    return this.cap;
  }

  length(): number {
    return this.count;
  }

  clear(): void {
    this.start = 0;
    this.count = 0;
  }

  appendOne(value: number): void {
    const end = (this.start + this.count) % this.cap;
    this.buf[end] = value;
    if (this.count < this.cap) {
      this.count++;
    } else {
      this.start = (this.start + 1) % this.cap;
    }
  }

  append(...values: number[]): void {
    if (!values || values.length === 0) return;
    for (const v of values) {
      this.appendOne(v);
    }
  }

  // Return ordered segments (oldest -> newest). May return 0, 1 or 2 segments.
  segments(): [] | [Float64Array] | [Float64Array, Float64Array] {
    if (this.count === 0) return [];
    const endIdx = this.start + this.count;
    if (endIdx <= this.cap) {
      // single contiguous segment
      return [this.buf.subarray(this.start, this.start + this.count)];
    }
    // wrapped: [start..cap) then [0..(endIdx%cap))
    const first = this.buf.subarray(this.start, this.cap);
    const second = this.buf.subarray(0, endIdx % this.cap);
    return [first, second];
  }
}

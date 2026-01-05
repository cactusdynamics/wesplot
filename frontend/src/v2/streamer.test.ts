import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Metadata } from "../types.js";
import {
  encodeMetadataMessage,
  encodeWSMessage,
  MessageTypeData,
  MessageTypeMetadata,
  MessageTypeStreamEnd,
  ProtocolVersion,
  type WSMessageData,
  type WSMessageMetadata,
  type WSMessageStreamEnd,
} from "./protocol.js";
import { Streamer, type StreamerCallbacks } from "./streamer.js";

// Helper to access private ws property
function getWebSocket(streamer: Streamer): MockWebSocket {
  const ws = (streamer as unknown as { _ws: MockWebSocket | null })._ws;
  if (!ws) {
    throw new Error("WebSocket is not initialized");
  }
  return ws;
}

// Mock WebSocket
class MockWebSocket {
  binaryType = "blob";
  readyState:
    | typeof WebSocket.CONNECTING
    | typeof WebSocket.OPEN
    | typeof WebSocket.CLOSING
    | typeof WebSocket.CLOSED = WebSocket.CONNECTING;
  url: string;

  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;

  private _errored = false;

  constructor(url: string) {
    this.url = url;
  }

  send(_data: string | ArrayBuffer | Blob): void {
    // Mock send
  }

  close(): void {
    this.readyState = WebSocket.CLOSED;
    if (this.onclose) {
      this.onclose(new CloseEvent("close"));
    }
  }

  // Helper method to simulate receiving a message
  simulateMessage(data: ArrayBuffer): void {
    if (this.onmessage) {
      this.onmessage(new MessageEvent("message", { data }));
    }
  }

  // Helper method to simulate error
  simulateError(): void {
    this._errored = true;
    if (this.onerror) {
      this.onerror(new Event("error"));
    }
  }

  // Helper method to simulate opening the connection
  simulateOpen(): void {
    if (!this._errored) {
      this.readyState = WebSocket.OPEN;
      if (this.onopen) {
        this.onopen(new Event("open"));
      }
    }
  }
}

// Helper to create test metadata message
function createTestMetadataMessage(
  options: { title?: string; columns?: string[] } = {},
): WSMessageMetadata {
  const { title = "Test", columns = ["Series1"] } = options;
  const metadata: Metadata = {
    WindowSize: 1000,
    XIsTimestamp: false,
    RelativeStart: false,
    WesplotOptions: {
      Title: title,
      Columns: columns,
      XLabel: "X",
      YLabel: "Y",
      YMin: undefined,
      YMax: undefined,
      YUnit: "",
      ChartType: "line",
    },
  };

  // Get the length in a cheating way so we can assert more easily later.
  const length = encodeMetadataMessage(metadata).length;

  return {
    Header: {
      Version: ProtocolVersion,
      Reserved: [0, 0],
      Type: MessageTypeMetadata,
      Length: length,
    },
    Payload: metadata,
  };
}

describe("Streamer", () => {
  let originalWebSocket: typeof WebSocket;

  beforeEach(() => {
    // Save original WebSocket
    originalWebSocket = globalThis.WebSocket;
    // Replace with mock
    globalThis.WebSocket = MockWebSocket as unknown as typeof WebSocket;
  });

  afterEach(() => {
    // Restore original WebSocket
    globalThis.WebSocket = originalWebSocket;
  });

  it("should throw error if windowSize is not a positive integer", () => {
    expect(() => new Streamer("ws://localhost", 0)).toThrow(
      "windowSize must be a positive integer",
    );
    expect(() => new Streamer("ws://localhost", -1)).toThrow(
      "windowSize must be a positive integer",
    );
    expect(() => new Streamer("ws://localhost", 1.5)).toThrow(
      "windowSize must be a positive integer",
    );
  });

  it("should reject if connect called twice after open", async () => {
    const streamer = new Streamer("ws://localhost/ws2", 1000);
    const connectPromise = streamer.connect();
    const mockWs = getWebSocket(streamer);
    mockWs.simulateOpen();
    await connectPromise;

    await expect(streamer.connect()).rejects.toThrow("Already connected");
    streamer.disconnect();
  });

  it("should reject if connect called twice before open", async () => {
    const streamer = new Streamer("ws://localhost/ws2", 1000);
    const connectPromise1 = streamer.connect();
    const connectPromise2 = streamer.connect(); // Should reject immediately
    await expect(connectPromise2).rejects.toThrow("Already connected");

    // Now simulate open for the first connection
    const mockWs = getWebSocket(streamer);
    mockWs.simulateOpen();
    await connectPromise1;
    streamer.disconnect();
  });

  it("should not call metadata after unregistering", async () => {
    const streamer = new Streamer("ws://localhost/ws2", 1000);
    const callbacks: StreamerCallbacks = {
      onMetadata: vi.fn(),
      onData: vi.fn(),
      onStreamEnd: vi.fn(),
      onError: vi.fn(),
    };

    // Register callbacks
    streamer.registerCallbacks(callbacks);

    // Connect and simulate events
    const connectPromise = streamer.connect();
    const mockWs = getWebSocket(streamer);
    mockWs.simulateOpen();
    await connectPromise;

    // Send metadata to trigger onMetadata
    const metadataMsg = createTestMetadataMessage();

    mockWs.simulateMessage(encodeWSMessage(metadataMsg).buffer as ArrayBuffer);

    // Verify callbacks were called
    expect(callbacks.onMetadata).toHaveBeenCalledTimes(1);

    // Unregister callbacks
    streamer.unregisterCallbacks(callbacks);

    // Reset mock call counts
    vi.clearAllMocks();

    // Send another metadata message
    mockWs.simulateMessage(encodeWSMessage(metadataMsg).buffer as ArrayBuffer);

    // Verify callbacks were NOT called after unregister
    expect(callbacks.onMetadata).not.toHaveBeenCalled();

    streamer.disconnect();
  });

  it("should handle METADATA message", async () => {
    const streamer = new Streamer("ws://localhost/ws2", 1000);
    const onMetadata = vi.fn();

    streamer.registerCallbacks({ onMetadata });
    const connectPromise = streamer.connect();
    const mockWs = getWebSocket(streamer);
    mockWs.simulateOpen();
    await connectPromise;

    const msg = createTestMetadataMessage({
      title: "Test Chart",
      columns: ["Series1", "Series2"],
    });

    const encoded = encodeWSMessage(msg);
    mockWs.simulateMessage(encoded.buffer as ArrayBuffer);

    expect(onMetadata).toHaveBeenCalledTimes(1);
    const receivedMetadata = onMetadata.mock.calls[0][0];
    expect(receivedMetadata.toJSON()).toEqual(msg.Payload);
    streamer.disconnect();
  });

  it("should handle DATA message", async () => {
    const streamer = new Streamer("ws://localhost/ws2", 1000);
    const onMetadata = vi.fn();
    const onData = vi.fn();

    streamer.registerCallbacks({ onMetadata, onData });
    const connectPromise = streamer.connect();
    const mockWs = getWebSocket(streamer);
    mockWs.simulateOpen();
    await connectPromise;

    // Send metadata first to create buffers
    const metadataMsg = createTestMetadataMessage();

    mockWs.simulateMessage(encodeWSMessage(metadataMsg).buffer as ArrayBuffer);

    // Send data message
    const dataMsg: WSMessageData = {
      Header: {
        Version: ProtocolVersion,
        Reserved: [0, 0],
        Type: MessageTypeData,
        // Length: 8 + Payload.Length * 16 (SeriesID(4) + Length(4) + X(8*Length) + Y(8*Length))
        Length: 56,
      },
      Payload: {
        SeriesID: 0,
        Length: 3,
        X: new Float64Array([1.0, 2.0, 3.0]),
        Y: new Float64Array([10.0, 20.0, 30.0]),
      },
    };

    mockWs.simulateMessage(encodeWSMessage(dataMsg).buffer as ArrayBuffer);

    expect(onData).toHaveBeenCalledTimes(1);
    expect(onData).toHaveBeenCalledWith(
      0,
      expect.any(Array),
      expect.any(Array),
    );

    // Verify the data in the segments
    const [seriesId, xSegments, ySegments] = onData.mock.calls[0];
    expect(seriesId).toBe(0);
    expect(xSegments).toHaveLength(1);
    expect(ySegments).toHaveLength(1);
    expect(Array.from(xSegments[0])).toEqual([1.0, 2.0, 3.0]);
    expect(Array.from(ySegments[0])).toEqual([10.0, 20.0, 30.0]);

    streamer.disconnect();
  });

  it("should handle series break (Length=0)", async () => {
    const streamer = new Streamer("ws://localhost/ws2", 1000);
    const onData = vi.fn();

    streamer.registerCallbacks({ onData });
    const connectPromise = streamer.connect();
    const mockWs = getWebSocket(streamer);
    mockWs.simulateOpen();
    await connectPromise;

    // Send metadata first
    const metadataMsg = createTestMetadataMessage();

    mockWs.simulateMessage(encodeWSMessage(metadataMsg).buffer as ArrayBuffer);

    // Send data message with Length=0 (series break)
    const breakMsg: WSMessageData = {
      Header: {
        Version: ProtocolVersion,
        Reserved: [0, 0],
        Type: MessageTypeData,
        // Length: 8 + Payload.Length * 16 (SeriesID(4) + Length(4) + X(8*Length) + Y(8*Length))
        Length: 8,
      },
      Payload: {
        SeriesID: 0,
        Length: 0,
        X: new Float64Array([]),
        Y: new Float64Array([]),
      },
    };

    mockWs.simulateMessage(encodeWSMessage(breakMsg).buffer as ArrayBuffer);

    expect(onData).toHaveBeenCalledTimes(1);
    const [_seriesId, xSegments, ySegments] = onData.mock.calls[0];

    // Should have inserted NaN
    expect(xSegments).toHaveLength(1);
    expect(ySegments).toHaveLength(1);
    expect(Number.isNaN(xSegments[0][0])).toBe(true);
    expect(Number.isNaN(ySegments[0][0])).toBe(true);

    streamer.disconnect();
  });

  it("should handle STREAM_END message", async () => {
    const streamer = new Streamer("ws://localhost/ws2", 1000);
    const onStreamEnd = vi.fn();

    streamer.registerCallbacks({ onStreamEnd });
    const connectPromise = streamer.connect();
    const mockWs = getWebSocket(streamer);
    mockWs.simulateOpen();
    await connectPromise;

    const streamEndMsg: WSMessageStreamEnd = {
      Header: {
        Version: ProtocolVersion,
        Reserved: [0, 0],
        Type: MessageTypeStreamEnd,
        Length: 0,
      },
      Payload: {
        Error: false,
        Msg: "Stream completed",
      },
    };

    mockWs.simulateMessage(encodeWSMessage(streamEndMsg).buffer as ArrayBuffer);

    expect(onStreamEnd).toHaveBeenCalledWith(false, "Stream completed");
    streamer.disconnect();
  });

  it("should handle WebSocket error", async () => {
    const streamer = new Streamer("ws://localhost/ws2", 1000);
    const onError = vi.fn();

    streamer.registerCallbacks({ onError });
    const connectPromise = streamer.connect();

    const mockWs = getWebSocket(streamer);
    mockWs.simulateError();

    await expect(connectPromise).rejects.toThrow("WebSocket connection failed");
    expect(onError).toHaveBeenCalled();
    expect(onError.mock.calls[0][0]).toBeInstanceOf(Error);
    streamer.disconnect();
  });

  it("should call onStreamEnd when connection closes without STREAM_END message", async () => {
    const streamer = new Streamer("ws://localhost/ws2", 1000);
    const onStreamEnd = vi.fn();

    streamer.registerCallbacks({ onStreamEnd });
    const connectPromise = streamer.connect();
    const mockWs = getWebSocket(streamer);
    mockWs.simulateOpen();
    await connectPromise;

    mockWs.close();

    expect(onStreamEnd).toHaveBeenCalledWith(
      true,
      "Connection closed: code 0, reason: none",
    );
  });

  it("should not call onStreamEnd twice if STREAM_END received before close", async () => {
    const streamer = new Streamer("ws://localhost/ws2", 1000);
    const onStreamEnd = vi.fn();

    streamer.registerCallbacks({ onStreamEnd });
    const connectPromise = streamer.connect();
    const mockWs = getWebSocket(streamer);
    mockWs.simulateOpen();
    await connectPromise;

    // Send STREAM_END
    const streamEndMsg: WSMessageStreamEnd = {
      Header: {
        Version: ProtocolVersion,
        Reserved: [0, 0],
        Type: MessageTypeStreamEnd,
        Length: 0,
      },
      Payload: {
        Error: false,
        Msg: "Complete",
      },
    };

    mockWs.simulateMessage(encodeWSMessage(streamEndMsg).buffer as ArrayBuffer);

    // Then close connection
    mockWs.close();

    // Should only be called once
    expect(onStreamEnd).toHaveBeenCalledTimes(1);
    expect(onStreamEnd).toHaveBeenCalledWith(false, "Complete");
  });

  it("should handle multiple callbacks", async () => {
    const streamer = new Streamer("ws://localhost/ws2", 1000);
    const onData1 = vi.fn();
    const onData2 = vi.fn();

    streamer.registerCallbacks({ onData: onData1 });
    streamer.registerCallbacks({ onData: onData2 });

    const connectPromise = streamer.connect();
    const mockWs = getWebSocket(streamer);
    mockWs.simulateOpen();
    await connectPromise;

    // Send metadata first to create buffers
    const metadataMsg = createTestMetadataMessage({ columns: ["S1"] });
    mockWs.simulateMessage(encodeWSMessage(metadataMsg).buffer as ArrayBuffer);

    // Send data message
    const dataMsg: WSMessageData = {
      Header: {
        Version: ProtocolVersion,
        Reserved: [0, 0],
        Type: MessageTypeData,
        Length: 56,
      },
      Payload: {
        SeriesID: 0,
        Length: 3,
        X: new Float64Array([1.0, 2.0, 3.0]),
        Y: new Float64Array([10.0, 20.0, 30.0]),
      },
    };

    mockWs.simulateMessage(encodeWSMessage(dataMsg).buffer as ArrayBuffer);

    expect(onData1).toHaveBeenCalledTimes(1);
    expect(onData2).toHaveBeenCalledTimes(1);
    const [seriesId1, xSegments1, ySegments1] = onData1.mock.calls[0];
    const [seriesId2, xSegments2, ySegments2] = onData2.mock.calls[0];
    expect(seriesId1).toBe(0);
    expect(seriesId2).toBe(0);
    expect(Array.from(xSegments1[0])).toEqual([1.0, 2.0, 3.0]);
    expect(Array.from(ySegments1[0])).toEqual([10.0, 20.0, 30.0]);
    expect(Array.from(xSegments2[0])).toEqual([1.0, 2.0, 3.0]);
    expect(Array.from(ySegments2[0])).toEqual([10.0, 20.0, 30.0]);

    // Unregister the second callback
    streamer.unregisterCallbacks({ onData: onData2 });

    // Send another data message
    mockWs.simulateMessage(encodeWSMessage(dataMsg).buffer as ArrayBuffer);

    // Verify only the first handler is called again
    expect(onData1).toHaveBeenCalledTimes(2);
    expect(onData2).toHaveBeenCalledTimes(1);

    streamer.disconnect();
  });

  it("should ignore data for unknown series", async () => {
    const streamer = new Streamer("ws://localhost/ws2", 1000);
    const onData = vi.fn();

    streamer.registerCallbacks({ onData });
    const connectPromise = streamer.connect();
    const mockWs = getWebSocket(streamer);
    mockWs.simulateOpen();
    await connectPromise;

    // Send data without metadata
    const dataMsg: WSMessageData = {
      Header: {
        Version: ProtocolVersion,
        Reserved: [0, 0],
        Type: MessageTypeData,
        Length: 0,
      },
      Payload: {
        SeriesID: 5,
        Length: 2,
        X: new Float64Array([1.0, 2.0]),
        Y: new Float64Array([10.0, 20.0]),
      },
    };

    mockWs.simulateMessage(encodeWSMessage(dataMsg).buffer as ArrayBuffer);

    expect(onData).not.toHaveBeenCalled();
    streamer.disconnect();
  });
});

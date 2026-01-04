import type { Metadata } from "../types.js";
import { CircularBuffer } from "./circular_buffer.js";
import {
  decodeWSMessage,
  MessageTypeData,
  MessageTypeMetadata,
  MessageTypeStreamEnd,
  type WSMessageData,
  type WSMessageMetadata,
  type WSMessageStreamEnd,
} from "./protocol.js";

export interface StreamerCallbacks {
  onMetadata?: (metadata: Metadata) => void;
  // Streamer owns per-series `CircularBuffer` instances (typed for float64).
  // When dispatching data, the Streamer will produce an ordered array of
  // `Float64Array` segments for X and Y respectively. In the common case
  // the array will contain a single segment; if the circular buffer wraps
  // two segments will be provided (end then begin). Charts must treat the
  // segments as concatenated in order.
  onData?: (
    seriesId: number,
    xSegments: Float64Array[],
    ySegments: Float64Array[],
  ) => void;
  onStreamEnd?: (error: boolean, message: string) => void;
  onError?: (error: Error) => void;
}

type OnMetadataCallback = NonNullable<StreamerCallbacks["onMetadata"]>;
type OnDataCallback = NonNullable<StreamerCallbacks["onData"]>;
type OnStreamEndCallback = NonNullable<StreamerCallbacks["onStreamEnd"]>;
type OnErrorCallback = NonNullable<StreamerCallbacks["onError"]>;

export class Streamer {
  private _ws: WebSocket | null = null;
  private _xBuffers: Map<number, CircularBuffer> = new Map();
  private _yBuffers: Map<number, CircularBuffer> = new Map();
  private _onMetadataCallbacks: Set<OnMetadataCallback> = new Set();
  private _onDataCallbacks: Set<OnDataCallback> = new Set();
  private _onStreamEndCallbacks: Set<OnStreamEndCallback> = new Set();
  private _onErrorCallbacks: Set<OnErrorCallback> = new Set();
  private _streamEndReceived = false;

  constructor(
    private wsUrl: string,
    private windowSize: number,
  ) {
    if (!Number.isInteger(windowSize) || windowSize <= 0) {
      throw new Error("windowSize must be a positive integer");
    }
  }

  // Register callbacks for stream events
  registerCallbacks(callbacks: StreamerCallbacks): void {
    if (callbacks.onMetadata)
      this._onMetadataCallbacks.add(callbacks.onMetadata);
    if (callbacks.onData) this._onDataCallbacks.add(callbacks.onData);
    if (callbacks.onStreamEnd)
      this._onStreamEndCallbacks.add(callbacks.onStreamEnd);
    if (callbacks.onError) this._onErrorCallbacks.add(callbacks.onError);
  }

  // Unregister callbacks so they stop getting events
  unregisterCallbacks(callbacks: StreamerCallbacks): void {
    if (callbacks.onMetadata)
      this._onMetadataCallbacks.delete(callbacks.onMetadata);
    if (callbacks.onData) this._onDataCallbacks.delete(callbacks.onData);
    if (callbacks.onStreamEnd)
      this._onStreamEndCallbacks.delete(callbacks.onStreamEnd);
    if (callbacks.onError) this._onErrorCallbacks.delete(callbacks.onError);
  }

  // Start streaming
  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this._ws) {
        reject(new Error("Already connected"));
        return;
      }

      this._streamEndReceived = false;
      this._ws = new WebSocket(this.wsUrl);
      this._ws.binaryType = "arraybuffer";

      this._ws.onopen = () => {
        resolve();
      };

      this._ws.onmessage = (event: MessageEvent) => {
        try {
          this._handleMessage(event.data);
        } catch (error) {
          this._invokeErrorCallbacks(
            error instanceof Error ? error : new Error(String(error)),
          );
        }
      };

      this._ws.onerror = () => {
        reject(new Error("WebSocket connection failed"));
        this._invokeErrorCallbacks(new Error("WebSocket error"));
        this._ws = null;
      };

      this._ws.onclose = (event: CloseEvent) => {
        if (!this._streamEndReceived) {
          const error = event.code !== 1000; // 1000 is normal closure
          const message = `Connection closed: code ${event.code}, reason: ${event.reason || "none"}`;
          this._invokeStreamEndCallbacks(error, message);
        }
        this._ws = null;
      };
    });
  }

  // Close connection
  disconnect(): void {
    if (this._ws) {
      this._ws.close();
      this._ws = null;
    }
  }

  private _handleMessage(data: ArrayBuffer): void {
    const buf = new Uint8Array(data);
    const msg = decodeWSMessage(buf);

    switch (msg.Header.Type) {
      case MessageTypeMetadata:
        this._handleMetadata(msg as WSMessageMetadata);
        break;
      case MessageTypeData:
        this._handleData(msg as WSMessageData);
        break;
      case MessageTypeStreamEnd:
        this._handleStreamEnd(msg as WSMessageStreamEnd);
        break;
      default:
        // Unknown message type - TypeScript doesn't know about this case
        msg.Header satisfies never;
        console.warn("Unknown message type:", msg);
        break;
    }
  }

  private _handleMetadata(msg: WSMessageMetadata): void {
    const metadata = msg.Payload;

    // Clear existing buffers
    this._xBuffers.clear();
    this._yBuffers.clear();

    // Create circular buffers for each series based on metadata
    const numSeries = metadata.WesplotOptions.Columns.length;
    for (let seriesId = 0; seriesId < numSeries; seriesId++) {
      this._xBuffers.set(seriesId, new CircularBuffer(this.windowSize));
      this._yBuffers.set(seriesId, new CircularBuffer(this.windowSize));
    }

    this._invokeMetadataCallbacks(metadata);
  }

  private _handleData(msg: WSMessageData): void {
    const { SeriesID, Length, X, Y } = msg.Payload;

    // Get buffers for this series
    const xBuffer = this._xBuffers.get(SeriesID);
    const yBuffer = this._yBuffers.get(SeriesID);

    if (!xBuffer || !yBuffer) {
      // Ignore data for unknown series
      console.warn(`Received data for unknown series ${SeriesID}, ignoring`);
      return;
    }

    // Handle series break (Length == 0)
    if (Length === 0) {
      // Insert NaN sentinel to indicate discontinuity
      xBuffer.appendOne(Number.NaN);
      yBuffer.appendOne(Number.NaN);
    } else {
      // Append data to circular buffers
      for (let i = 0; i < Length; i++) {
        xBuffer.appendOne(X[i]);
        yBuffer.appendOne(Y[i]);
      }
    }

    // Get segments and dispatch to callbacks
    const xSegments = xBuffer.segments();
    const ySegments = yBuffer.segments();

    this._invokeDataCallbacks(SeriesID, xSegments, ySegments);
  }

  private _handleStreamEnd(msg: WSMessageStreamEnd): void {
    const payload = msg.Payload;
    this._streamEndReceived = true;
    this._invokeStreamEndCallbacks(payload.Error, payload.Msg);
  }

  private _invokeMetadataCallbacks(metadata: Metadata): void {
    for (const cb of this._onMetadataCallbacks) {
      cb(metadata);
    }
  }

  private _invokeDataCallbacks(
    seriesId: number,
    xSegments: Float64Array[],
    ySegments: Float64Array[],
  ): void {
    for (const cb of this._onDataCallbacks) {
      cb(seriesId, xSegments, ySegments);
    }
  }

  private _invokeStreamEndCallbacks(error: boolean, message: string): void {
    if (error) {
      console.error(`Stream end error: ${message}`);
    } else {
      console.log(`Stream end: ${message}`);
    }

    for (const cb of this._onStreamEndCallbacks) {
      cb(error, message);
    }
  }

  private _invokeErrorCallbacks(error: Error): void {
    console.error(error);
    for (const cb of this._onErrorCallbacks) {
      cb(error);
    }
  }
}

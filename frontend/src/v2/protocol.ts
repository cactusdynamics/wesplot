import type { Metadata } from "./types.js";

// Detect host endianness
export const isLittleEndian = (() => {
  const buffer = new ArrayBuffer(2);
  new DataView(buffer).setInt16(0, 256, true /* littleEndian */);
  // Int16Array uses the platform's endianness.
  return new Int16Array(buffer)[0] === 256;
})();

// Protocol constants
export const ProtocolVersion = 1;

export const MessageTypeData = 0x01;
export const MessageTypeMetadata = 0x02;
export const MessageTypeStreamEnd = 0x03;

export const EnvelopeHeaderSize = 8;

type EnvelopeHeaderBase = {
  Version: number;
  Reserved: [number, number];
  Length: number; // payload length in bytes
};

export type EnvelopeHeaderData = EnvelopeHeaderBase & {
  Type: typeof MessageTypeData;
};

export type EnvelopeHeaderMetadata = EnvelopeHeaderBase & {
  Type: typeof MessageTypeMetadata;
};

export type EnvelopeHeaderStreamEnd = EnvelopeHeaderBase & {
  Type: typeof MessageTypeStreamEnd;
};

export type EnvelopeHeader =
  | EnvelopeHeaderData
  | EnvelopeHeaderMetadata
  | EnvelopeHeaderStreamEnd;

export interface DataMessage {
  SeriesID: number; // uint32
  Length: number; // uint32 number of pairs
  X: Float64Array;
  Y: Float64Array;
  toJSON?(): DataMessage;
}

export interface StreamEndMessage {
  Error: boolean;
  Msg: string;
  toJSON?(): StreamEndMessage;
}

export interface WSMessageData {
  Header: EnvelopeHeaderData;
  Payload: DataMessage;
}

export interface WSMessageMetadata {
  Header: EnvelopeHeaderMetadata;
  Payload: Metadata;
}

export interface WSMessageStreamEnd {
  Header: EnvelopeHeaderStreamEnd;
  Payload: StreamEndMessage;
}

export type WSMessage = WSMessageData | WSMessageMetadata | WSMessageStreamEnd;

// Implementation classes with lazy decoding
export class DataMessageLazyObjLittleEndianHost implements DataMessage {
  private view: DataView;

  constructor(private buf: Uint8Array) {
    this.view = new DataView(this.buf.buffer, this.buf.byteOffset);
  }

  get SeriesID(): number {
    return this.view.getUint32(0, true);
  }

  get Length(): number {
    return this.view.getUint32(4, true);
  }

  get X(): Float64Array {
    const length = this.Length;
    const offset = 8;
    return new Float64Array(
      this.buf.buffer,
      this.buf.byteOffset + offset,
      length,
    );
  }

  get Y(): Float64Array {
    const length = this.Length;
    const offset = 8 + length * 8;
    return new Float64Array(
      this.buf.buffer,
      this.buf.byteOffset + offset,
      length,
    );
  }

  toJSON(): DataMessage {
    return {
      SeriesID: this.SeriesID,
      Length: this.Length,
      X: new Float64Array(this.X),
      Y: new Float64Array(this.Y),
    };
  }
}

export class DataMessageLazyObjBigEndianHost implements DataMessage {
  private view: DataView;

  constructor(private buf: Uint8Array) {
    this.view = new DataView(this.buf.buffer, this.buf.byteOffset);
  }

  get SeriesID(): number {
    return this.view.getUint32(0, true);
  }

  get Length(): number {
    return this.view.getUint32(4, true);
  }

  get X(): Float64Array {
    const length = this.Length;
    const X = new Float64Array(length);
    let offset = 8;
    for (let i = 0; i < length; i++) {
      X[i] = this.view.getFloat64(offset, true);
      offset += 8;
    }
    return X;
  }

  get Y(): Float64Array {
    const length = this.Length;
    const Y = new Float64Array(length);
    let offset = 8 + length * 8;
    for (let i = 0; i < length; i++) {
      Y[i] = this.view.getFloat64(offset, true);
      offset += 8;
    }
    return Y;
  }

  toJSON(): DataMessage {
    return {
      SeriesID: this.SeriesID,
      Length: this.Length,
      X: this.X,
      Y: this.Y,
    };
  }
}

// Select the appropriate implementation based on host endianness
let DataMessageLazyObj = isLittleEndian
  ? DataMessageLazyObjLittleEndianHost
  : DataMessageLazyObjBigEndianHost;

// For testing purposes
export function setDataMessageBigEndianDecode(bigEndian: boolean) {
  DataMessageLazyObj = bigEndian
    ? DataMessageLazyObjBigEndianHost
    : DataMessageLazyObjLittleEndianHost;
}

class MetadataLazyObj implements Metadata {
  private view: DataView;
  private _parsed?: Metadata;

  constructor(private buf: Uint8Array) {
    this.view = new DataView(this.buf.buffer, this.buf.byteOffset);
  }

  private _getParsedObj(): Metadata {
    if (!this._parsed) {
      const jsonLength = this.view.getUint32(0, true);
      const jsonBytes = this.buf.slice(4, 4 + jsonLength);
      const decoder = new TextDecoder();
      const json = decoder.decode(jsonBytes);
      this._parsed = JSON.parse(json) as Metadata;
    }
    return this._parsed;
  }

  get WindowSize(): number {
    return this._getParsedObj().WindowSize;
  }

  get XIsTimestamp(): boolean {
    return this._getParsedObj().XIsTimestamp;
  }

  get RelativeStart(): boolean {
    return this._getParsedObj().RelativeStart;
  }

  get WesplotOptions() {
    return this._getParsedObj().WesplotOptions;
  }

  toJSON(): Metadata {
    return {
      WindowSize: this.WindowSize,
      XIsTimestamp: this.XIsTimestamp,
      RelativeStart: this.RelativeStart,
      WesplotOptions: this.WesplotOptions,
    };
  }
}

class StreamEndMessageLazyObj implements StreamEndMessage {
  private view: DataView;
  private _parsed?: StreamEndMessage;

  constructor(private buf: Uint8Array) {
    this.view = new DataView(this.buf.buffer, this.buf.byteOffset);
  }

  private _getParsedObj(): StreamEndMessage {
    if (!this._parsed) {
      const jsonLength = this.view.getUint32(0, true);
      const jsonBytes = this.buf.slice(4, 4 + jsonLength);
      const decoder = new TextDecoder();
      const json = decoder.decode(jsonBytes);
      const parsed = JSON.parse(json) as { error: boolean; msg: string };
      this._parsed = { Error: parsed.error, Msg: parsed.msg };
    }
    return this._parsed;
  }

  get Error(): boolean {
    return this._getParsedObj().Error;
  }

  get Msg(): string {
    return this._getParsedObj().Msg;
  }

  toJSON(): StreamEndMessage {
    return {
      Error: this.Error,
      Msg: this.Msg,
    };
  }
}

// Encode/decode envelope header
export function encodeEnvelopeHeader(env: EnvelopeHeader): Uint8Array {
  const buf = new Uint8Array(EnvelopeHeaderSize);
  const view = new DataView(buf.buffer);
  view.setUint8(0, env.Version & 0xff);
  view.setUint8(1, env.Reserved[0] & 0xff);
  view.setUint8(2, env.Reserved[1] & 0xff);
  view.setUint8(3, env.Type & 0xff);
  view.setUint32(4, env.Length >>> 0, true);
  return buf;
}

export function decodeEnvelopeHeader(buf: Uint8Array): EnvelopeHeader {
  if (buf.length < EnvelopeHeaderSize) {
    throw new Error(
      `buffer too short: expected at least ${EnvelopeHeaderSize} bytes, got ${buf.length}`,
    );
  }
  const view = new DataView(buf.buffer, buf.byteOffset, buf.byteLength);
  const version = view.getUint8(0);
  const reserved: [number, number] = [view.getUint8(1), view.getUint8(2)];
  const type = view.getUint8(3);
  const length = view.getUint32(4, true);

  return {
    Version: version,
    Reserved: reserved,
    Type: type as EnvelopeHeader["Type"], // Override this even tho it can be wrong!
    Length: length,
  };
}

// DATA message encoding/decoding
export function encodeDataMessage(msg: DataMessage): Uint8Array {
  if (msg.X.length !== msg.Y.length) {
    throw new Error(
      `X and Y arrays must have same length: X=${msg.X.length}, Y=${msg.Y.length}`,
    );
  }
  if (msg.Length !== msg.X.length) {
    throw new Error(
      `Length field (${msg.Length}) doesn't match array length (${msg.X.length})`,
    );
  }

  const payloadSize = 8 + msg.Length * 8 * 2;
  const buf = new Uint8Array(payloadSize);
  const view = new DataView(buf.buffer);
  view.setUint32(0, msg.SeriesID >>> 0, true); // force 32-bit unsigned
  view.setUint32(4, msg.Length >>> 0, true);
  let offset = 8;
  // write X
  for (let i = 0; i < msg.Length; i++) {
    view.setFloat64(offset, msg.X[i], true);
    offset += 8;
  }
  // write Y
  for (let i = 0; i < msg.Length; i++) {
    view.setFloat64(offset, msg.Y[i], true);
    offset += 8;
  }
  return buf;
}

export function decodeDataMessage(buf: Uint8Array): DataMessage {
  if (buf.length < 8) {
    throw new Error(
      `buffer too short for DATA message: expected at least 8 bytes, got ${buf.length}`,
    );
  }
  const view = new DataView(buf.buffer, buf.byteOffset, buf.byteLength);
  const length = view.getUint32(4, true);
  const expectedSize = 8 + length * 8 * 2;
  if (buf.length !== expectedSize) {
    throw new Error(
      `buffer size mismatch: expected ${expectedSize} bytes for ${length} pairs, got ${buf.length}`,
    );
  }
  return new DataMessageLazyObj(buf);
}

// METADATA encoding/decoding (JSON length + JSON bytes)
export function encodeMetadataMessage(metadata: Metadata): Uint8Array {
  const json = JSON.stringify(metadata);
  const encoder = new TextEncoder();
  const jsonBytes = encoder.encode(json);
  const payloadSize = 4 + jsonBytes.length;
  const buf = new Uint8Array(payloadSize);
  const view = new DataView(buf.buffer);
  view.setUint32(0, jsonBytes.length >>> 0, true);
  buf.set(jsonBytes, 4);
  return buf;
}

export function decodeMetadataMessage(buf: Uint8Array): Metadata {
  if (buf.length < 4) {
    throw new Error(
      `buffer too short for METADATA message: expected at least 4 bytes, got ${buf.length}`,
    );
  }
  const view = new DataView(buf.buffer, buf.byteOffset, buf.byteLength);
  const jsonLength = view.getUint32(0, true);
  const expectedSize = 4 + jsonLength;
  if (buf.length !== expectedSize) {
    throw new Error(
      `buffer size mismatch: expected ${expectedSize} bytes, got ${buf.length}`,
    );
  }
  return new MetadataLazyObj(buf);
}

// STREAM_END encoding/decoding
export function encodeStreamEndMessage(msg: StreamEndMessage): Uint8Array {
  const json = JSON.stringify({ error: msg.Error, msg: msg.Msg });
  const encoder = new TextEncoder();
  const jsonBytes = encoder.encode(json);
  const payloadSize = 4 + jsonBytes.length;
  const buf = new Uint8Array(payloadSize);
  const view = new DataView(buf.buffer);
  view.setUint32(0, jsonBytes.length >>> 0, true);
  buf.set(jsonBytes, 4);
  return buf;
}

export function decodeStreamEndMessage(buf: Uint8Array): StreamEndMessage {
  if (buf.length < 4) {
    throw new Error(
      `buffer too short for STREAM_END message: expected at least 4 bytes, got ${buf.length}`,
    );
  }
  const view = new DataView(buf.buffer, buf.byteOffset, buf.byteLength);
  const jsonLength = view.getUint32(0, true);
  const expectedSize = 4 + jsonLength;
  if (buf.length !== expectedSize) {
    throw new Error(
      `buffer size mismatch: expected ${expectedSize} bytes, got ${buf.length}`,
    );
  }
  return new StreamEndMessageLazyObj(buf);
}

const payloadEncoders: {
  [K in EnvelopeHeader["Type"]]: (msg: WSMessage["Payload"]) => Uint8Array;
} = {
  [MessageTypeData]: encodeDataMessage as (
    msg: WSMessage["Payload"],
  ) => Uint8Array,
  [MessageTypeMetadata]: encodeMetadataMessage as (
    msg: WSMessage["Payload"],
  ) => Uint8Array,
  [MessageTypeStreamEnd]: encodeStreamEndMessage as (
    msg: WSMessage["Payload"],
  ) => Uint8Array,
};

const payloadDecoders: {
  [K in EnvelopeHeader["Type"]]: (
    payloadBytes: Uint8Array,
  ) => WSMessage["Payload"];
} = {
  [MessageTypeData]: decodeDataMessage,
  [MessageTypeMetadata]: decodeMetadataMessage,
  [MessageTypeStreamEnd]: decodeStreamEndMessage,
};

// WSMessage encode/decode
export function encodeWSMessage<T extends WSMessage>(msg: T): Uint8Array {
  const encoder = payloadEncoders[msg.Header.Type];
  if (!encoder) {
    console.error("unknown message type", msg.Header);
    throw new Error("unknown message type");
  }

  const payload = encoder(msg.Payload);

  // set header length
  msg.Header.Length = payload.length;
  const header = encodeEnvelopeHeader(msg.Header);
  const full = new Uint8Array(header.length + payload.length);
  full.set(header, 0);
  full.set(payload, header.length);
  return full;
}

export function decodeWSMessage(buf: Uint8Array): WSMessage {
  const env = decodeEnvelopeHeader(buf);
  const expectedSize = EnvelopeHeaderSize + env.Length;
  if (buf.length < expectedSize) {
    throw new Error(
      `buffer too short: expected ${expectedSize} bytes (header + payload), got ${buf.length}`,
    );
  }

  const payloadBytes = buf.slice(
    EnvelopeHeaderSize,
    EnvelopeHeaderSize + env.Length,
  );
  const decoder = payloadDecoders[env.Type];
  if (!decoder) {
    throw new Error(`unknown message type: 0x${env.Type.toString(16)}`);
  }
  const payload = decoder(payloadBytes);
  return { Header: env, Payload: payload } as WSMessage;
}

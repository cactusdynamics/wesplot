import { afterEach, beforeAll, describe, expect, it } from "vitest";
import type { DataMessage, StreamEndMessage, WSMessage } from "./protocol.js";
import {
  decodeDataMessage,
  decodeMetadataMessage,
  decodeStreamEndMessage,
  decodeWSMessage,
  encodeDataMessage,
  encodeWSMessage,
  isLittleEndian,
  MessageTypeData,
  MessageTypeMetadata,
  MessageTypeStreamEnd,
  ProtocolVersion,
  setDataMessageBigEndianDecode,
} from "./protocol.js";

import type { Metadata } from "./types.js";

type MetadataWithToJSON = Metadata & {
  toJSON?(): Metadata;
};

describe("Protocol", () => {
  describe("encodeWSMessage and decodeWSMessage round-trips", () => {
    it("round-trips DATA message", () => {
      const dataMsg: DataMessage = {
        SeriesID: 0,
        Length: 2,
        X: new Float64Array([1.0, 2.0]),
        Y: new Float64Array([10.0, 20.0]),
      };

      const wsMsg: WSMessage = {
        Header: {
          Version: ProtocolVersion,
          Reserved: [0, 0],
          Type: MessageTypeData,
          Length: 8 + dataMsg.Length * 16, // SeriesID(4) + Length(4) + X(8*Length) + Y(8*Length)
        },
        Payload: dataMsg,
      };

      const encoded = encodeWSMessage(wsMsg);
      const decoded = decodeWSMessage(encoded);

      expect(decoded.Header.Version).toBe(ProtocolVersion);
      expect(decoded.Header.Type).toBe(MessageTypeData);

      expect((decoded.Payload as DataMessage).toJSON?.()).toEqual(dataMsg);
    });

    it("round-trips METADATA message", () => {
      const metadata: Metadata = {
        WindowSize: 1000,
        XIsTimestamp: true,
        RelativeStart: false,
        WesplotOptions: {
          Title: "Test Chart",
          Columns: ["A", "B"],
          XLabel: "Time",
          YLabel: "Value",
          YUnit: "units",
          ChartType: "line",
        },
      };

      const wsMsg: WSMessage = {
        Header: {
          Version: ProtocolVersion,
          Reserved: [0, 0],
          Type: MessageTypeMetadata,
          Length: 0,
        },
        Payload: metadata,
      };

      const encoded = encodeWSMessage(wsMsg);
      const decoded = decodeWSMessage(encoded);

      expect(decoded.Header.Type).toBe(MessageTypeMetadata);

      expect((decoded.Payload as MetadataWithToJSON).toJSON?.()).toEqual(
        metadata,
      );
    });

    it("round-trips STREAM_END message", () => {
      const streamEnd: StreamEndMessage = {
        Error: true,
        Msg: "Error occurred",
      };

      const wsMsg: WSMessage = {
        Header: {
          Version: ProtocolVersion,
          Reserved: [0, 0],
          Type: MessageTypeStreamEnd,
          Length: 0,
        },
        Payload: streamEnd,
      };

      const encoded = encodeWSMessage(wsMsg);
      const decoded = decodeWSMessage(encoded);

      expect(decoded.Header.Type).toBe(MessageTypeStreamEnd);

      expect((decoded.Payload as StreamEndMessage).toJSON?.()).toEqual(
        streamEnd,
      );
    });

    it("round-trips empty DATA message (series break)", () => {
      const dataMsg: DataMessage = {
        SeriesID: 5,
        Length: 0,
        X: new Float64Array([]),
        Y: new Float64Array([]),
      };

      const wsMsg: WSMessage = {
        Header: {
          Version: ProtocolVersion,
          Reserved: [0, 0],
          Type: MessageTypeData,
          Length: 8 + dataMsg.Length * 16,
        },
        Payload: dataMsg,
      };

      const encoded = encodeWSMessage(wsMsg);
      const decoded = decodeWSMessage(encoded);

      const decodedData = decoded.Payload as DataMessage;
      expect(decodedData.SeriesID).toBe(dataMsg.SeriesID);
      expect(decodedData.Length).toBe(0);
      expect(decodedData.X.length).toBe(0);
      expect(decodedData.Y.length).toBe(0);
    });

    it("preserves reserved bytes", () => {
      const dataMsg: DataMessage = {
        SeriesID: 1,
        Length: 1,
        X: new Float64Array([5.0]),
        Y: new Float64Array([10.0]),
      };

      const wsMsg: WSMessage = {
        Header: {
          Version: ProtocolVersion,
          Reserved: [0xab, 0xcd],
          Type: MessageTypeData,
          Length: 8 + dataMsg.Length * 16,
        },
        Payload: dataMsg,
      };

      const encoded = encodeWSMessage(wsMsg);
      const decoded = decodeWSMessage(encoded);

      expect(decoded.Header.Reserved).toEqual([0xab, 0xcd]);
    });

    it("handles special float values", () => {
      const dataMsg: DataMessage = {
        SeriesID: 0,
        Length: 5,
        X: new Float64Array([0.0, -0.0, Infinity, -Infinity, NaN]),
        Y: new Float64Array([1.0, 2.0, 3.0, 4.0, 5.0]),
      };

      const wsMsg: WSMessage = {
        Header: {
          Version: ProtocolVersion,
          Reserved: [0, 0],
          Type: MessageTypeData,
          Length: 8 + dataMsg.Length * 16,
        },
        Payload: dataMsg,
      };

      const encoded = encodeWSMessage(wsMsg);
      const decoded = decodeWSMessage(encoded);

      const decodedData = decoded.Payload as DataMessage;
      expect(decodedData.Length).toBe(5);
      expect(decodedData.X[0]).toBe(0.0);
      expect(decodedData.X[1]).toBe(-0.0); // Note: -0 becomes 0 in Float64Array
      expect(decodedData.X[2]).toBe(Infinity);
      expect(decodedData.X[3]).toBe(-Infinity);
      expect(Number.isNaN(decodedData.X[4])).toBe(true);
    });
  });

  describe("decodeWSMessage error cases", () => {
    it("throws on buffer too short for header", () => {
      const buf = new Uint8Array([1, 2, 3]);
      expect(() => decodeWSMessage(buf)).toThrow("buffer too short");
    });

    it("throws on buffer too short for payload", () => {
      // Create a header claiming a large payload but provide insufficient data
      const header = new Uint8Array(8);
      const view = new DataView(header.buffer);
      view.setUint8(0, ProtocolVersion);
      view.setUint8(3, MessageTypeData);
      view.setUint32(4, 1000, true); // Claim 1000 bytes payload

      expect(() => decodeWSMessage(header)).toThrow("buffer too short");
    });

    it("throws on unknown message type", () => {
      const header = new Uint8Array(8);
      const view = new DataView(header.buffer);
      view.setUint8(0, ProtocolVersion);
      view.setUint8(3, 0xff); // Invalid type
      view.setUint32(4, 0, true); // No payload

      expect(() => decodeWSMessage(header)).toThrow("unknown message type");
    });
  });

  describe("encodeWSMessage error cases", () => {
    it("throws on unknown message type", () => {
      const wsMsg = {
        Header: {
          Version: ProtocolVersion,
          Reserved: [0, 0] as [number, number],
          Type: 0xff,
          Length: 0,
        },
        Payload: {},
      } as unknown as WSMessage;

      expect(() => encodeWSMessage(wsMsg)).toThrow("unknown message type");
    });
  });

  describe("encodeDataMessage error cases", () => {
    it("throws when X and Y arrays have different lengths", () => {
      const dataMsg: DataMessage = {
        SeriesID: 0,
        Length: 2,
        X: new Float64Array([1.0, 2.0]),
        Y: new Float64Array([10.0]), // Different length
      };

      expect(() => encodeDataMessage(dataMsg)).toThrow(
        "X and Y arrays must have same length",
      );
    });

    it("throws when Length field doesn't match array length", () => {
      const dataMsg: DataMessage = {
        SeriesID: 0,
        Length: 3, // Wrong length
        X: new Float64Array([1.0, 2.0]),
        Y: new Float64Array([10.0, 20.0]),
      };

      expect(() => encodeDataMessage(dataMsg)).toThrow(
        "Length field (3) doesn't match array length (2)",
      );
    });
  });

  describe("decodeDataMessage error cases", () => {
    it("throws on buffer too short", () => {
      const buf = new Uint8Array([1, 2, 3]);
      expect(() => decodeDataMessage(buf)).toThrow(
        "buffer too short for DATA message",
      );
    });

    it("throws on buffer size mismatch", () => {
      const buf = new Uint8Array(16); // 8 header + 8 for 1 pair, but claim 2 pairs
      const view = new DataView(buf.buffer);
      view.setUint32(0, 0, true); // SeriesID
      view.setUint32(4, 2, true); // Length = 2, but buffer only has space for 1

      expect(() => decodeDataMessage(buf)).toThrow("buffer size mismatch");
    });
  });

  describe("decodeMetadataMessage error cases", () => {
    it("throws on buffer too short", () => {
      const buf = new Uint8Array([1, 2, 3]);
      expect(() => decodeMetadataMessage(buf)).toThrow(
        "buffer too short for METADATA message",
      );
    });

    it("throws on buffer size mismatch", () => {
      const buf = new Uint8Array(8); // 4 for length + 4 for data, but claim more
      const view = new DataView(buf.buffer);
      view.setUint32(0, 10, true); // jsonLength = 10, but buffer only has 4 bytes after

      expect(() => decodeMetadataMessage(buf)).toThrow("buffer size mismatch");
    });
  });

  describe("decodeStreamEndMessage error cases", () => {
    it("throws on buffer too short", () => {
      const buf = new Uint8Array([1, 2, 3]);
      expect(() => decodeStreamEndMessage(buf)).toThrow(
        "buffer too short for STREAM_END message",
      );
    });

    it("throws on buffer size mismatch", () => {
      const buf = new Uint8Array(8); // 4 for length + 4 for data, but claim more
      const view = new DataView(buf.buffer);
      view.setUint32(0, 10, true); // jsonLength = 10, but buffer only has 4 bytes after

      expect(() => decodeStreamEndMessage(buf)).toThrow("buffer size mismatch");
    });
  });
});

describe("Big Endian Implementation Data Decoder", () => {
  beforeAll(() => {
    setDataMessageBigEndianDecode(true);
  });

  afterEach(() => {
    setDataMessageBigEndianDecode(!isLittleEndian);
  });

  it("round-trips DATA message with big endian implementation", () => {
    const dataMsg: DataMessage = {
      SeriesID: 0,
      Length: 2,
      X: new Float64Array([1.0, 2.0]),
      Y: new Float64Array([10.0, 20.0]),
    };

    const wsMsg: WSMessage = {
      Header: {
        Version: ProtocolVersion,
        Reserved: [0, 0],
        Type: MessageTypeData,
        Length: 8 + dataMsg.Length * 16, // SeriesID(4) + Length(4) + X(8*Length) + Y(8*Length)
      },
      Payload: dataMsg,
    };

    const encoded = encodeWSMessage(wsMsg);
    const decoded = decodeWSMessage(encoded);

    expect(decoded.Header.Version).toBe(ProtocolVersion);
    expect(decoded.Header.Type).toBe(MessageTypeData);

    expect((decoded.Payload as DataMessage).toJSON?.()).toEqual(dataMsg);
  });
});

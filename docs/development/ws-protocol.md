# Wesplot Binary Envelope Protocol Specification

**Version:** 1.0
**Date:** 30 December 2025
**Endpoint:** `/ws2`

## Overview

The wesplot WS protocol is a binary envelope protocol for streaming XY series data over WebSockets. It provides:

- **Binary encoding** for improved performance and reduced parsing overhead
- **Envelope framing** to support multiple message types in a single stream
- **Multi-X/Y data model** allowing series with different X-value sampling
- **Inline control messages** for metadata, errors, and stream status
- **Protocol versioning** for future extensibility

## Design Principles

1. **Simple and Efficient:** Custom binary format optimized for streaming numeric data
2. **Zero External Dependencies:** No protobuf, msgpack, or other libraries required
3. **Self-Describing:** Each message includes type and length information
4. **Extensible:** Message type field allows future protocol additions
5. **Language-Agnostic:** Binary format can be implemented in any language

## Wire Format

### Envelope Structure

Every message sent over `/ws2` follows this envelope structure:

```
+------------------+------------------------+------------------+------------------------+
| Version (1 byte) | Reserved (2 bytes)     | Type (1 byte)    | Length (4 bytes, LE)   |
+------------------+------------------------+------------------+------------------------+
| Payload (variable length)                                                            |
+--------------------------------------------------------------------------------------+
```

**Field Descriptions:**

- **Version (1 byte):** Protocol version number. Current version is `1`.
- **Reserved (2 bytes):** Reserved for future use. Can be any value (forward compatible).
- **Type (1 byte):** Message type identifier (see Message Types section).
- **Length (4 bytes, Little Endian):** Length of payload in bytes (uint32).
- **Payload (variable):** Message-specific payload data.

**Total Header Size:** 8 bytes (aligned)

- All multi-byte integers use **Little Endian (LE)**
- IEEE 754 double-precision floats (8 bytes each) for numeric data

**Rationale:** Little Endian is the native byte order for most browsers and modern processors, providing better performance without byte-swapping overhead. Float64 provides sufficient precision for large integer values (up to 2^53) without precision loss.

## Message Types

### Type Constants

| Type      | Name           | Description                          |
|-----------|----------------|--------------------------------------|
| 0x01      | DATA           | Time-series data with X/Y arrays     |
| 0x02      | METADATA       | Stream metadata and configuration    |
| 0x03      | STREAM_END     | End of stream indicator with error   |
| 0x04-0xFF | RESERVED       | Reserved for future use              |

### Message Type: DATA (0x01)

Transmits time-series data for a **single series**. Each DATA message contains data for one series only, identified by series index. DATA messages are deltas. Each DATA message contains new data points that should be **appended** to the client's accumulated state for that series. The message does NOT contain the full series data. The initial DATA message will contain all data points buffered by the server.

#### Payload Structure

```
+----------------------------+
| SeriesID (4 bytes LE)      |  Series index (0-based)
+----------------------------+
| Length (4 bytes LE)        |  Number of X/Y pairs
+----------------------------+
| X[0] (8 bytes)             |  First X value (float64)
| X[1] (8 bytes)             |  Second X value (float64)
| ...                        |
| X[Length-1] (8 bytes)      |  Last X value
+----------------------------+
| Y[0] (8 bytes)             |  First Y value (float64)
| Y[1] (8 bytes)             |  Second Y value (float64)
| ...                        |
| Y[Length-1] (8 bytes)      |  Last Y value
+----------------------------+
```

**Field Descriptions:**

- **SeriesID:** Zero-based index of the series (uint32)
  - Corresponds to index in `WesplotOptions.Columns` array from METADATA
  - Example: SeriesID=0 refers to first series, SeriesID=1 refers to second series
- **Length:** Number of X/Y data point pairs in this message (uint32)
- **X[i]:** X value at index i (float64, 8 bytes)
- **Y[i]:** Y value at index i (float64, 8 bytes)

**Constraints:**

- SeriesID must be valid (0 ≤ SeriesID < number of series in METADATA)
- Length must equal number of X values and Y values (paired data)
- Length ≥ 0
- X and Y arrays must have exactly `Length` elements each
- **Length=0 (empty message) indicates a series break:**
  - Signals discontinuity in the data stream
  - For line charts: points before and after this message should NOT be connected
  - No X or Y data follows (payload is just SeriesID + Length=0)
  - Next DATA message for same series starts a new segment

**Append-Only Semantics:**

- First DATA message after connection: May contain historical buffer (all accumulated data)
- Subsequent DATA messages: Contain only NEW data points
- Client must APPEND received data to existing series data

Example flow:

```
METADATA: 2 series ("CPU", "Memory")
DATA(SeriesID=0, Length=100): Historical CPU data (points 0-99)
DATA(SeriesID=1, Length=100): Historical Memory data (points 0-99)
DATA(SeriesID=0, Length=10): New CPU data (append points 100-109)
DATA(SeriesID=1, Length=10): New Memory data (append points 100-109)
DATA(SeriesID=0, Length=5): More CPU data (append points 110-114)
```

#### DATA byte example

Series 0 with 3 new data points:

```
SeriesID: 0
Length: 3
X: [1.0, 2.0, 3.0]
Y: [10.5, 20.3, 15.7]
```

**Byte Layout:**
```
00 00 00 00                            # SeriesID = 0
03 00 00 00                            # Length = 3
00 00 00 00 00 00 F0 3F                # X[0] = 1.0 (float64)
00 00 00 00 00 00 00 40                # X[1] = 2.0 (float64)
00 00 00 00 00 00 08 40                # X[2] = 3.0 (float64)
00 00 00 00 00 00 25 40                # Y[0] = 10.5 (float64)
CD CC CC CC CC 4C 34 40                # Y[1] = 20.3 (float64)
66 66 66 66 66 66 2F 40                # Y[2] = 15.7 (float64)
```

**Decoding Strategy:**

1. Read SeriesID (4 bytes)
2. Read Length (4 bytes)
3. Read Length float64 values for X array
4. Read Length float64 values for Y array
5. Append X/Y pairs to the series identified by SeriesID

**Updating Multiple Series:**

To update multiple series, send multiple DATA messages:
```
DATA(SeriesID=0, Length=5): 5 new points for series 0
DATA(SeriesID=1, Length=5): 5 new points for series 1
DATA(SeriesID=2, Length=5): 5 new points for series 2
```

Series can be updated independently:
```
DATA(SeriesID=0, Length=10): Update series 0 only
DATA(SeriesID=0, Length=5): Update series 0 again (series 1 unchanged)
DATA(SeriesID=1, Length=3): Now update series 1
```

### Message Type: METADATA (0x02)

Transmits stream configuration and metadata. Sent once at connection establishment.

#### Payload Structure

```
+---------------------------+
| JSON Length (4 bytes LE)  |  Length of JSON string in bytes
+---------------------------+
| JSON Data (UTF-8 string)  |  JSON-encoded Metadata object
+---------------------------+
```

**Field Descriptions:**

- **JSON Length:** Byte length of the JSON string (uint32)
- **JSON Data:** UTF-8 encoded JSON string

**JSON Schema:**

The JSON payload matches the Go `Metadata` struct:

```json
{
  "WindowSize": 1000,
  "XIsTimestamp": true,
  "RelativeStart": false,
  "WesplotOptions": {
    "Title": "System Metrics",
    "Columns": ["CPU Usage", "Memory Usage"],
    "XLabel": "Time (s)",
    "YLabel": "Value",
    "YMin": null,
    "YMax": null,
    "YUnit": "%",
    "ChartType": "line"
  }
}
```

**Field Descriptions:**

- **WindowSize:** Number of data points to display in rolling window (0 = infinite)
- **XIsTimestamp:** Whether X values represent Unix timestamps
- **RelativeStart:** Whether to display X values relative to first value
- **WesplotOptions.Title:** Chart title
- **WesplotOptions.Columns:** Array of series names/labels
- **WesplotOptions.XLabel:** X-axis label
- **WesplotOptions.YLabel:** Y-axis label
- **WesplotOptions.YMin:** Minimum Y-axis value (null = auto)
- **WesplotOptions.YMax:** Maximum Y-axis value (null = auto)
- **WesplotOptions.YUnit:** Unit string for Y values
- **WesplotOptions.ChartType:** Chart type ("line", "bar", etc.)

**Example:**
```json
{
  "WindowSize": 1000,
  "XIsTimestamp": true,
  "RelativeStart": false,
  "WesplotOptions": {
    "Title": "System Metrics",
    "Columns": ["CPU", "Memory"],
    "XLabel": "Time",
    "YLabel": "Usage %",
    "YUnit": "%",
    "ChartType": "line"
  }
}
```

**Rationale:**

- METADATA is only sent once per connection (performance not critical)
- JSON provides flexibility and easy frontend parsing
- Reuses existing `Metadata` Go struct
- `Columns` field provides series names corresponding to DATA message series order

### Message Type: STREAM_END (0x03)

Indicates end of data stream. Sent once when input source is exhausted or on error. This replaces the separate ERROR message type.

#### Payload Structure

```
+---------------------------+
| JSON Length (4 bytes LE)  |  Length of JSON string in bytes
+---------------------------+
| JSON Data (UTF-8 string)  |  JSON-encoded end message
+---------------------------+
```

**Field Descriptions:**

- **JSON Length:** Byte length of the JSON string (uint32)
- **JSON Data:** UTF-8 encoded JSON string

**JSON Schema:**

```json
{
  "error": false,
  "msg": "Stream completed successfully"
}
```

or

```json
{
  "error": true,
  "msg": "Input stream read error: EOF"
}
```

**Field Descriptions:**

- **error:** Boolean indicating whether stream ended due to error (true = error, false = clean termination)
- **msg:** Human-readable termination message (can be empty string)

**Examples:**

**Clean End:**
```json
{"error": false, "msg": ""}
```

**Clean End with Message:**
```json
{"error": false, "msg": "Stream completed successfully"}
```

**Error End:**
```json
{"error": true, "msg": "Failed to read from stdin: unexpected EOF"}
```

**Behavior:**

- STREAM_END is the final message sent
- Empty `msg` indicates clean termination
- Non-empty `msg` indicates error or informational termination message
- Connection should be closed after STREAM_END
- Client should stop expecting further messages
- WebSocket close frame should follow STREAM_END

**Rationale:**

- Any error results in stream termination (no non-fatal errors)
- JSON provides flexibility for future fields

## Connection Lifecycle

### Connection Establishment

1. Client connects to `/ws2` endpoint via WebSocket
2. Server accepts connection with binary frame support
3. Server immediately sends METADATA message (0x02)
4. Server sends historical data buffer as DATA messages (one per series) if available
   - Example: `DATA(SeriesID=0, Length=100)`, `DATA(SeriesID=1, Length=100)`
5. Server streams live data as DATA messages (delta/append-only)

### Normal Operation

1. Server sends DATA messages as new data arrives (one per series that has updates)
2. Client appends data to corresponding series
3. Client processes messages based on type field

### Connection Termination

1. Server sends STREAM_END message (0x03) when stream completes or error occurs
2. Server sends WebSocket close frame (status 1000 = Normal Closure)
3. Connection closes

### Error Handling

- **All errors are fatal:** Send STREAM_END with error message, close connection
- **Client disconnect:** Clean up server-side resources, no message sent

## Implementation Considerations

### Server-Side (Go)

1. Use `binary.LittleEndian` for encoding/decoding integers
2. Use `math.Float64bits` and `math.Float64frombits` for float64 encoding
3. Use binary message WebSocket frame type
4. Use `json.Marshal` for METADATA and STREAM_END payloads
5. Buffer DATA messages for batching (optional, performance optimization)
6. Flush buffers on timeout or capacity threshold

### Client-Side (TypeScript/JavaScript)

TODO

## Protocol Versioning

### Current Version: 1

**Version Field:** First byte of every message

### Future Versions

- If protocol changes are needed, increment version number
- Server should support multiple versions if backward compatibility is needed
- Client should check version field and handle accordingly
- Version negotiation can be added via query parameters or separate handshake

### Backward Compatibility Strategy

- New message types can be added without breaking version 1
- Clients should ignore unknown message types (warn when encountered for the first time)
- Payload structure for existing types must not change in version 1
- Breaking changes require version increment

## Example Message Sequences

### Typical Session

```
1. Client connects to ws://host/ws2
2. Server → METADATA (0x02) - JSON with 2 series: ["CPU", "Memory"]
3. Server → DATA (SeriesID=0, Length=100) - historical CPU data
4. Server → DATA (SeriesID=1, Length=100) - historical Memory data
5. Server → DATA (SeriesID=0, Length=10) - new CPU data (append)
6. Server → DATA (SeriesID=1, Length=10) - new Memory data (append)
7. Server → DATA (SeriesID=0, Length=5) - more CPU data (append)
8. Server → DATA (SeriesID=1, Length=5) - more Memory data (append)
9. ...
10. Server → STREAM_END (0x03) - {"error": false, "msg": ""} (clean termination)
11. WebSocket closes
```

### Session with Error

```
1. Client connects
2. Server → METADATA (0x02)
3. Server → DATA (SeriesID=0, Length=50) - historical
4. Server → DATA (SeriesID=1, Length=50) - historical
5. Server → DATA (SeriesID=0, Length=10) - live
6. Server → STREAM_END (0x03) - {"error": true, "msg": "Failed to read stdin: I/O error"}
7. WebSocket closes
```

### Session with Series Break

```
1. Client connects
2. Server → METADATA (0x02) - JSON with 1 series: ["Temperature"]
3. Server → DATA (SeriesID=0, Length=50) - first segment
4. Server → DATA (SeriesID=0, Length=0) - SERIES BREAK (no data, signals discontinuity)
5. Server → DATA (SeriesID=0, Length=50) - second segment (not connected to first)
6. Server → STREAM_END (0x03) - {"error": false, "msg": ""}
7. WebSocket closes
```

## Performance Characteristics

### Message Overhead

- **Header:** 8 bytes per message (aligned)
- **DATA message:** 8 bytes (SeriesID, Length) + 16N bytes (N point pairs, float64)
  - Total per message: 16 + 16N bytes
- **METADATA:** Variable (JSON encoding, typically 200-500 bytes)
- **STREAM_END:** Variable (JSON encoding, typically 50-200 bytes)

## Testing Requirements

### Unit Tests (Protocol Layer)

- [ ] Encode/decode round-trips for all message types
- [ ] Handle empty arrays and zero-length strings
- [ ] Handle large payloads (1000+ points)
- [ ] Reject malformed messages (invalid lengths, unknown types)
- [ ] Handle edge cases (Length=0 for series break, invalid SeriesID)
- [ ] Verify byte order (Little Endian)
- [ ] Verify float64 encoding accuracy
- [ ] Verify JSON encoding for METADATA and STREAM_END
- [ ] Verify STREAM_END includes both error boolean and msg fields
- [ ] Verify single-series DATA message encoding/decoding
- [ ] Verify header alignment (8 bytes)
- [ ] Verify decoder ignores reserved bytes (any value accepted)

### Integration Tests (WebSocket Layer)

- [ ] METADATA sent on connection with correct JSON
- [ ] Historical buffer transmitted correctly
- [ ] Live data streaming works
- [ ] Multi-client broadcasting
- [ ] Series break (Length=0) handled correctly
- [ ] STREAM_END sent on completion with error=false
- [ ] STREAM_END sent on error with error=true and error message
- [ ] Connection closes cleanly
- [ ] `/ws` endpoint unaffected (regression test)

## References

- **Architecture:** See `docs/development/architecture.md`
- **DataRow Structure:** See `metadata.go`

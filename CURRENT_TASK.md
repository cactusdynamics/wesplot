# Wesplot Backend Refactoring: /ws2 Binary Envelope Protocol

**Date Started:** 30 December 2025

## Context and Rationale

We are implementing a major refactoring of the wesplot backend's websocket protocol to address three critical limitations:

### Problems with Current /ws Endpoint

1. **JSON Protocol Performance Issues**
   - Current protocol streams JSON-encoded arrays of `DataRow` objects
   - JSON parsing can become a bottleneck when piping high-frequency data
   - Each message requires full JSON deserialization on the frontend
   - No binary encoding support for efficient data transfer

2. **Limited Message Types**
   - Protocol only supports `DataRow` type (X: float64, Ys: []float64)
   - No envelope/framing to encapsulate different message types
   - Metadata and errors require separate HTTP endpoints (`/metadata`, `/errors`)
   - Cannot send control messages inline with data stream
   - No protocol versioning or extensibility

3. **Single X Value Limitation**
   - Current `DataRow` assigns one X value to multiple Y values
   - Eliminates possibility of having series with offset X values
   - Cannot represent data where different series have different X sampling
   - Limitation extends throughout data broadcaster architecture

### Refactoring Strategy

This is a **phased refactoring**. We cannot change everything at once, so we're starting with:

**Phase 1 (Current):** Create new `/ws2` endpoint with binary envelope protocol
- Implement binary wire format with message type framing
- Support multiple X arrays and corresponding Y arrays
- Inline metadata and error messages in websocket stream
- Leave existing `/ws` endpoint untouched (backward compatibility)
- Leave upstream components (DataBroadcaster, DataReader) unchanged
- Leave frontend unchanged for now

**Future Phases:**
- Refactor DataBroadcaster to support multi-X/Y data model
- Update frontend to consume `/ws2` protocol
- Migrate or deprecate `/ws` endpoint
- Update data readers and parsing logic

## Implementation TODO List

### Phase 1: /ws2 Binary Envelope Protocol Backend

- [x] **Step 1:** Record task context and TODO list in CURRENT_TASK.md

- [x] **Step 2:** Design and document binary envelope protocol
  - [x] Create docs/development/ws2-protocol.md
  - [x] Define wire format (header structure, length fields, payload encoding)
  - [x] Define message types: Data, Metadata, Error, StreamEnd
  - [x] Document multi-X/Y data payload format
  - [x] Document metadata message format
  - [x] Document error message format
  - [x] Document protocol versioning strategy
  - [x] Include examples and byte diagrams
  - [x] **REQUEST REVIEW AFTER THIS STEP**

- [x] **Step 3:** Implement core protocol encoding/decoding
  - [x] Create ws_protocol.go with message type constants
  - [x] Implement envelope message encoder/decoder
  - [x] Implement multi-X/Y data payload encoder/decoder
  - [x] Implement metadata message encoder/decoder
  - [x] Implement stream-end message encoder/decoder

- [x] **Step 4:** Write comprehensive unit tests for protocol
  - [x] Create ws_protocol_test.go
  - [x] Test encoding/decoding round-trips
  - [x] Test edge cases: empty arrays, single values, large payloads
  - [x] Test malformed data handling
  - [x] Test all message types
  - [x] Verify 100% line coverage with `make test COVERAGE=1`

- [x] **Step 5:** Add /ws2 handler in http_server.go
  - [x] Implement handleWebSocket2() function
  - [x] Accept websocket connection with binary frames
  - [x] Send metadata envelope on connection
  - [x] Register channel with DataBroadcaster
  - [x] Transform single-X DataRow to multi-X format (duplicate X per Y)
  - [x] Stream data using binary protocol
  - [x] Send error envelopes on stream issues
  - [x] Send stream-end envelope on completion
  - [x] Add route: `s.mux.HandleFunc("/ws2", s.handleWebSocket2)`

- [x] **Step 6:** Write integration tests for /ws2
  - [x] Add tests to http_server_test.go
  - [x] Test binary message parsing
  - [x] Test metadata delivery on connect
  - [x] Test data streaming with multi-X format
  - [x] Test multi-client broadcasting
  - [x] Test stream-end envelope
  - [x] Test error envelope propagation
  - [x] Add regression test: verify /ws still works unchanged

- [x] **Step 7**: Write a test client in Go
  - [x] Implement a command line based decoder that can decode the envelope message and internal data from the websocket for testing purposes

- [x] **Step 8:** Final validation
  - [x] Run `make test COVERAGE=1` - achieved 83.2% coverage overall
  - [x] Run `make lint` - no errors or warnings
  - [x] Verify all /ws tests still pass (no regression)
  - [x] Verify all /ws2 tests pass

## Backward Compatibility

- `/ws` endpoint remains completely unchanged
- Both endpoints coexist and can be used simultaneously
- Frontend can choose which endpoint to connect to
- No breaking changes to existing functionality

## Important Notes for Sub-Agents

**Testing Policy:**
- All code changes must have comprehensive unit tests
- Code coverage must be 100% - verify with `make test COVERAGE=1`
- Think through edge cases and document why they matter
- Test failures must have good error messages

**Completion Policy:**
- No temporary TODOs or placeholders in code
- Run `make test` and `make lint` before marking tasks complete
- If blocked, explain why and ask user for guidance

**Progress Tracking:**
- Update this TODO list as work progresses
- Check off completed items with [x]
- Mark current item as IN PROGRESS
- Keep context section up to date

## Follow-Up Work (Out of Scope for Phase 1)

- [ ] Update frontend TypeScript to decode binary protocol
- [ ] Update Player class to handle envelope messages
- [ ] Refactor DataBroadcaster for native multi-X/Y support
- [ ] Update data readers to parse multi-X data
- [ ] Migrate frontend to use /ws2
- [ ] Deprecate /ws endpoint
- [ ] Update documentation for end users

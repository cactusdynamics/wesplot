# Phase 2: Frontend Rewrite for Multi-Series Support

**Date Started:** 1 January 2026

## Context and Rationale

Phase 1 successfully implemented the `/ws2` binary envelope protocol backend, enabling multi-X/Y data streaming. However, the frontend remains tied to the original JSON-based `/ws` endpoint, supporting only single-X shared across all series and a single plot display.

### Problems with Current Frontend

1. **Single Plot Limitation**
   - Frontend displays only one chart/plot visual element instead of supporting split views

2. **Shared X-Value Assumption**
   - Assumes all series share the same X values as it uses the /ws JSON instead of the /ws2 binary protocol
   - Cannot handle series with different X sampling or offsets

3. **Tight Coupling and Poor Architecture**
   - Player component handles both streaming and UI logic
   - Deep coupling between Player and WesplotChart
   - Difficult to extend or test independently

4. **Performance Concerns**
   - Not optimized for high-frequency data streaming
   - Potential memory allocation issues in vanilla JS/TypeScript

### Phase 2 Refactoring Strategy

This is a **major frontend rewrite** to support multi-series with independent X values. We will:

**Create v2 Frontend:**
- New entrypoint: `v2.html`
- TypeScript code in `src/v2/` directory
- Maintain existing `frontend/` for backward compatibility

**New Architecture:**
- **Streamer Component:** Connects to `/ws2`, decodes binary protocol, manages streaming
  - Registers arbitrary callbacks for data events
  - Handles metadata, data, and stream-end messages
  - Optimized for performance (minimize allocations)
- **Chart Component:** Reusable vanilla JS component for rendering charts
  - Can be instantiated multiple times, each time taking in a different container element to take ownership over
  - Configurable per chart (series selection, display options)
  - Supports multiple series per chart with different X values

**Incremental Approach:**
- Start with single chart showing multiple series
- Future: Support multiple chart instances
- Maintain vanilla JS for performance and simplicity

**Performance Focus:**
- Minimize object creation and copying
- Use efficient data structures for streaming data
- Batch updates where possible

## Implementation TODO List

### Phase 2: Frontend Rewrite

- [x] **Step 1:** Document new frontend architecture in docs/development/frontend-architecture.md
  - [x] Describe Streamer component responsibilities
  - [x] Describe Chart component API and lifecycle
  - [x] Document data flow between components
  - [x] Include diagrams for component interactions
  - [x] **REQUEST REVIEW AFTER THIS STEP**

- [x] **Step 2:** Set up v2 frontend structure
  - [x] Create `v2.html` as new entrypoint
  - [x] Create `src/v2/` directory for TypeScript code
  - [x] Set up testing/benchmark infrastructure for TypeScript and add coverage/linting commands

- [ ] **Step 3:** Implement Streamer component
  - [ ] Create `src/v2/protocol.ts` implementing envelope parsing and message decoding (mirror `ws_protocol.go` behavior)
  - [ ] Add unit tests for protocol decode/encode round-trips and malformed data handling
  - [ ] Create `src/v2/circular_buffer.ts` implementing a typed `CircularBuffer<Float64Array>` abstraction with `append`, `reserve`, and a method to produce ordered `Float64Array` segment views (1 or 2 segments when wrapped)
  - [ ] Add unit tests and benchmarks for `CircularBuffer` (wrap and non-wrap cases, performance)

- [ ] **Step 4:** Implement Streamer using `CircularBuffer`
  - [ ] Create `src/v2/streamer.ts`
  - [ ] Implement WebSocket connection to `/ws2` and use `protocol.ts` for message decoding
  - [ ] On METADATA: parse JSON, create per-series `CircularBuffer` instances keyed by seriesId
  - [ ] On DATA: append into the `CircularBuffer` (insert `NaN` sentinel for Length==0 breaks), and dispatch ordered `Float64Array` segment arrays via callbacks
  - [ ] On STREAM_END: notify callbacks and close connection
  - [ ] Support callback registration/deregistration and the segment-array `onData` API
  - [ ] Optimize for low allocation (reuse buffers, avoid copying on hot path)
  - [ ] Add comprehensive tests and benchmarks for Streamer behavior (including integration with `CircularBuffer`)

 - [ ] **Step 5:** Implement Chart component
  - [ ] Consider approaches for testing visual elements for AI agents
  - [ ] Create `src/v2/chart.ts`
  - [ ] Define Chart API (constructor options: series IDs, display config)
  - [ ] Integrate with Chart.js for rendering
  - [ ] Handle data updates from Streamer callbacks
  - [ ] Support multiple series with independent X values
  - [ ] Implement efficient data appending (no full re-renders)
  - [ ] Add basic configuration (colors, labels, etc.)

- [ ] **Step 6:** Create v2 main application
  - [ ] Create `src/v2/main.ts`
  - [ ] Initialize Streamer and connect to `/ws2`
  - [ ] Create one or more Chart instances
  - [ ] Register chart update callbacks with Streamer
  - [ ] Handle connection lifecycle (connect, stream end, errors)

- [ ] **Step 7:** Add comprehensive tests for v2 components
  - [ ] Integration tests for v2 app (end-to-end streaming)
  - [ ] Performance tests (memory usage, frame rates)
  - [ ] Ensure 100% coverage where possible

- [ ] **Step 8:** Update build and deployment
  - [ ] Update Makefile to build v2 frontend
  - [ ] Ensure v2.html is served by backend
  - [ ] Test v2 with live data streaming
  - [ ] Verify no regressions in original frontend

- [ ] **Step 9:** Final validation and documentation
  - [ ] Run all tests (backend and frontend)
  - [ ] Update user documentation for v2 features
  - [ ] Mark Phase 2 complete

## Backward Compatibility

- Original `frontend/` remains unchanged and functional
- `/ws` endpoint continues to work
- Users can choose v1 or v2 frontend via URL
- No breaking changes to backend API

## Important Notes for Sub-Agents

**Testing Policy:**
- All code changes must have comprehensive unit tests
- Performance-critical code should include benchmarks
- Test edge cases: empty data, single points, high frequency
- Test failure scenarios: WebSocket disconnects, malformed messages

**Completion Policy:**
- No temporary TODOs or placeholders in code
- Run tests and lint before marking tasks complete
- If blocked, explain why and ask user for guidance

**Progress Tracking:**
- Update this TODO list as work progresses
- Check off completed items with [x]
- Mark current item as IN PROGRESS
- Keep context section up to date

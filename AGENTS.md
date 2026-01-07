# AI agent instructions for wesplot

Wesplot is a project that gets live data as input and pipes it into the browser via a websocket connection so it can be plotted. The technology stack is a Golang backend coupled with a vanilla TypeScript frontend.

The current focus of the project is to pipe the Go binary's stdin to one or more browser tabs where it is plotted via chart.js.

## Documentation

Read these if not sure about data flow or architecture.

- Architecture and file structure: docs/development/architecture.md
- Data protocol between processes: docs/development/ws-protocol.md
- Frontend architecture: docs/development/frontend-architecture.md

## Build and test instructions

VERY VERY IMPORTANT TO FOLLOW THE FOLLOWING: Use these EXACT commands CHARACTER BY CHARACTER as they will be auto approved and do not use other commands as they will need approval and will be slower. If you do not use one of the commands below, ALWAYS explain why you need the deviation before running it.

Backend instructions

- Build the backend binary with the frontend: `make prod`
- Lint the backend code: `make backend-lint`
- Run all backend tests: `make backend-test`
- Run all the backend tests with coverage: `make backend-test COVERAGE=1`
- If the user wants to run a subset of the tests: `go test -run <TestNameRegexp> ./...` (not auto approved but may be useful).

Frontend instructions (need to `cd frontend` first, which might already been done in the open terminal).

- Build the frontend code: `npm run build`
- Type Check and lint the frontend code and apply any fixes: `npm run lint:write`
- Run all frontend tests: `npm run test`
  - DO NOT PASS `--silent` to this command as its output has already been minimized.
- Run all frontend tests with coverage: `npm run test:coverage`
- Run the frontend benchmarks: `npm run benchmark`
- Run a specific frontend test: `npm run test -- <filename> -t <pattern>` (not auto approved but may be useful).

## Coding rules

All rules below applies unless told otherwise by user prompts.

### Top level rules (always follow)

- ALWAYS run tests and lint to make sure all errors, warnings, test failures, and other issues are resolved before marking the task as completed.

### Test policy

- Add sufficient test coverage for code changes. Think of all the possible edge cases and comment inline in the tests on why these cases matter.
- Code coverage should be 100% (but some error paths might be near-impossible to test, so they can be skipped).
- Unit tests should be in `<file>_test.go` as per normal Go convention, `<file>.test.ts` for frontend, and `<file>.bench.ts` for frontend benchmarks.
- Test failures should be accompanied with good error messages for debugging.
- Follow test commands exactly as stated above where possible.

### Completion policy

- No temporary TODOs, placeholders, and workarounds in the code.
- If truly cannot fix the issues or blocked, state why and ask user for help.

### Documentation policy

- Think carefully if the architecture has been changed. Before changing the architecture, present the plans and ask for feedback from the user. Once the change is approved and implemented, update docs/development/architecture.md.

# AI agent instructions for wesplot

Wesplot is a project that gets live data as input and pipes it into the browser via a websocket connection so it can be plotted. The technology stack is a Golang backend coupled with a vanilla TypeScript frontend.

The current focus of the project is to pipe the Go binary's stdin to one or more browser tabs where it is plotted via chart.js.

The architecture and file structure of the project is documented in [docs/development/architecture.md](./docs/development/architecture.md). Read this file and read the referenced files if not sure about data flow or architecture.

## Build and test instructions

- Building the binary: `make prod` (see Makefile if necessary) which creates `build/wesplot`.
- Lint the code with `make lint`
- Run all tests: `make test`
- Run all tests and check for code coverage: `make test COVERAGE=1`

## Coding rules

All rules below applies unless told otherwise by user prompts.

### Top level rules (always follow)

- ALWAYS run tests and lint to make sure all errors, warnings, test failures, and other issues are resolved before marking the task as completed.

### Test policy

- Add sufficient test coverage for code changes. Think of all the possible edge cases and comment inline in the tests on why these cases matter.
- Code coverage should be 100%. Check with `make test COVERAGE=1`.
- Unit tests should be in `<file>_test.go` as per normal Go convention.
- Test failures should be accompanied with good error messages for debugging.

### Completion policy

- No temporary TODOs, placeholders, and workarounds in the code.
- If truly cannot fix the issues or blocked, state why and ask user for help.

### Documentation policy

- Think carefully if the architecture has been changed. Before changing the architecture, present the plans and ask for feedback from the user. Once the change is approved and implemented, update docs/development/architecture.md.

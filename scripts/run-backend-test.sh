#!/bin/bash

set -e

TIMEOUT_DURATION=${TIMEOUT_DURATION:-30s}

# Explicitly list packages to avoid including main binaries in coverage reports
PACKAGES=". ./cmd/wesplot-ws-reader"

if [ "$COVERAGE" = "1" ]; then
  echo "Running backend tests (with coverage)..."
  timeout -v ${TIMEOUT_DURATION} go test -coverprofile=${COVERAGE_FILE} ${PACKAGES}
  echo
  echo "Backend coverage summary:"
  go tool cover -func=${COVERAGE_FILE}
  go tool cover -html ${COVERAGE_FILE} -o ${COVERAGE_FILE}.html
else
  echo "Running backend tests (no coverage)..."
  timeout -v ${TIMEOUT_DURATION} go test ${PACKAGES}
fi

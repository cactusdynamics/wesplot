#!/bin/bash

set -xe

PACKAGES=.

if [ "$COVERAGE" = "1" ]; then
  echo "Running backend tests (with coverage)..."
  go test -coverprofile=${COVERAGE_FILE} ${PACKAGES}
  echo
  echo "Backend coverage summary:"
  go tool cover -func=${COVERAGE_FILE}
  go tool cover -html ${COVERAGE_FILE} -o ${COVERAGE_FILE}.html
else
  echo "Running backend tests (no coverage)..."
  go test ${PACKAGES}
fi

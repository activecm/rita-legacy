#!/bin/sh

echo "Running static test - gofmt"
GOFMT_FILES=$(gofmt -l .)
if [ -n "${GOFMT_FILES}" ]; then
  printf >&2 "gofmt failed for the following files:\n%s\n\nplease run 'gofmt -w .' on your changes before committing.\n" "${GOFMT_FILES}"
  TEST_STATUS=FAIL
fi

echo "Running static test - golint"
GOLINT_ERRORS=$(golint ./...)
if [ -n "${GOLINT_ERRORS}" ]; then
  printf >&2 "golint failed for the following reasons:\n%s\n\nplease run 'golint ./...' on your changes before committing.\n" "${GOLINT_ERRORS}"
  TEST_STATUS=FAIL
fi

echo "Running static test - go tool vet"
GOVET_ERRORS=$(go tool vet $(find . -name '*.go' | grep -v '/vendor/') 2>&1)
if [ -n "${GOVET_ERRORS}" ]; then
  printf >&2 "go vet failed for the following reasons:\n%s\n\nplease run \"go tool vet \$(find . -name '*.go' | grep -v '/vendor/')\" on your changes before committing.\n" "${GOVET_ERRORS}"
  TEST_STATUS=FAIL
fi

if [ "$TEST_STATUS" = "FAIL" ]; then
  exit 1
fi

#!/usr/bin/env sh

go test -coverpkg=./... -coverprofile=coverage.out ./...

go tool cover -func=coverage.out

min_coverage=45
coverage=$(go tool cover -func=coverage.out | grep -e '^total:' | awk '{print $3}' | sed 's/%//g')
if [ $(echo "${coverage} < ${min_coverage}" | bc) -ne 0 ]; then
  echo "ERROR: Coverage (${coverage}) is lower than minimum (${min_coverage})"
  exit $min_coverage
fi

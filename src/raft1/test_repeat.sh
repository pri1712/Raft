#!/bin/bash
for i in {1..10}; do
  echo "========== Run $i =========="
  output=$(go test -run $1 2>&1)
  echo "$output"
  failed=$(echo "$output" | grep -E "FAIL\s+[a-zA-Z0-9_/]+\.Test[^\s]+")

  if [ -n "$failed" ]; then
    echo "Run $i failed tests:"
    echo "$failed"
  else
    echo "Run $i passed all 3B tests"
  fi
done
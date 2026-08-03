#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: run_stress.sh <output-json>" >&2
  exit 1
fi

output=$1
duration_seconds=${STRESS_DURATION_SECONDS:-1800}
deadline=$((SECONDS + duration_seconds))
iterations=0

commands=(
  "GOWORK=off go test -race . ./internal/appserver ./internal/router ./internal/runtimebin"
  "GOWORK=off go test -race -run 'TestTurnHandleSteerAndInterrupt|TestClientStartInitializes|TestRouterGoalIsolationAcrossThreads|TestConcurrentInstallConverges' -count=20 . ./internal/appserver ./internal/router ./internal/runtimebin"
)

while (( SECONDS < deadline )); do
  for cmd in "${commands[@]}"; do
    bash -lc "$cmd"
  done
  iterations=$((iterations + 1))
done

mkdir -p "$(dirname "$output")"
cat >"$output" <<JSON
{
  "durationSeconds": $duration_seconds,
  "iterations": $iterations,
  "commands": [
    "${commands[0]}",
    "${commands[1]}"
  ]
}
JSON

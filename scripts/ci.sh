#!/usr/bin/env bash
# Local CI. Thin wrapper around `make prepush`, which is the single source
# of truth for the local gate (race + shuffle tests, vet, lint, example
# builds). Kept as a script so existing muscle memory and any external
# callers keep working.
set -euo pipefail

cd "$(dirname "$0")/.."
exec make prepush

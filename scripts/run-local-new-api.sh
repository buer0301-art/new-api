#!/usr/bin/env bash
set -euo pipefail

cd /Users/chuxin/data/claude/new-api
exec /opt/homebrew/bin/go run . --log-dir ./logs

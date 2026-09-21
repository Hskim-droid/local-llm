#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

python3 harness.py init
python3 harness.py doctor
python3 harness.py session-start --agent codex

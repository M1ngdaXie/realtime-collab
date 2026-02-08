#!/usr/bin/env bash
set -euo pipefail

# Simple helper to run k6 and capture pprof during peak load.
# Usage:
#   PPROF_BASE=http://localhost:8080/debug/pprof \
#   K6_SCRIPT=loadtest/loadtest_prod.js \
#   SLEEP_BEFORE=40 \
#   CPU_SECONDS=30 \
#   ./loadtest/run_with_pprof.sh

PPROF_BASE="${PPROF_BASE:-http://localhost:8080/debug/pprof}"
K6_SCRIPT="${K6_SCRIPT:-loadtest/loadtest_prod.js}"
SLEEP_BEFORE="${SLEEP_BEFORE:-40}"
CPU_SECONDS="${CPU_SECONDS:-30}"
OUT_DIR="${OUT_DIR:-loadtest/pprof}"

STAMP="$(date +%Y%m%d_%H%M%S)"
RUN_DIR="${OUT_DIR}/${STAMP}"

mkdir -p "${RUN_DIR}"

echo "==> Starting k6: ${K6_SCRIPT}"
k6 run "${K6_SCRIPT}" &
K6_PID=$!

echo "==> Waiting ${SLEEP_BEFORE}s before capturing pprof..."
sleep "${SLEEP_BEFORE}"

echo "==> Capturing profiles into ${RUN_DIR}"
curl -sS "${PPROF_BASE}/profile?seconds=${CPU_SECONDS}" -o "${RUN_DIR}/cpu.pprof"
curl -sS "${PPROF_BASE}/mutex" -o "${RUN_DIR}/mutex.pprof"
curl -sS "${PPROF_BASE}/block" -o "${RUN_DIR}/block.pprof"
curl -sS "${PPROF_BASE}/goroutine?debug=2" -o "${RUN_DIR}/goroutine.txt"

echo "==> Waiting for k6 to finish (pid ${K6_PID})..."
wait "${K6_PID}"

echo "==> Done. Profiles saved in ${RUN_DIR}"
echo "    Inspect with:"
echo "      go tool pprof -top ${RUN_DIR}/cpu.pprof"
echo "      go tool pprof -top ${RUN_DIR}/mutex.pprof"
echo "      go tool pprof -top ${RUN_DIR}/block.pprof"

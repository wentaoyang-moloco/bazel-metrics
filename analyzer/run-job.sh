#!/bin/bash
set -e

# Configuration (can be overridden by environment variables)
REPO_URL="${REPO_URL:-https://github.com/user/repo.git}"
REPO_BRANCH="${REPO_BRANCH:-main}"
GCS_BUCKET="${GCS_BUCKET:-bazel-metrics-data}"
RUN_BENCHMARKS="${RUN_BENCHMARKS:-false}"
MAX_BENCHMARKS="${MAX_BENCHMARKS:-5}"

echo "=== Bazel Metrics Analyzer Job ==="
echo "Repo: $REPO_URL"
echo "Branch: $REPO_BRANCH"
echo "GCS Bucket: $GCS_BUCKET"
echo "Run Benchmarks: $RUN_BENCHMARKS"

# Create working directory
WORK_DIR="/tmp/repo"
mkdir -p "$WORK_DIR"

# Clone the repository
echo ""
echo "=== Cloning repository ==="
if [ -n "$GIT_TOKEN" ]; then
    # Use token authentication if provided
    REPO_WITH_TOKEN=$(echo "$REPO_URL" | sed "s|https://|https://${GIT_TOKEN}@|")
    git clone --depth 1 --branch "$REPO_BRANCH" "$REPO_WITH_TOKEN" "$WORK_DIR"
else
    git clone --depth 1 --branch "$REPO_BRANCH" "$REPO_URL" "$WORK_DIR"
fi

echo ""
echo "=== Running analyzer ==="
BENCHMARK_FLAG=""
if [ "$RUN_BENCHMARKS" = "true" ]; then
    BENCHMARK_FLAG="--benchmark --max-benchmarks=$MAX_BENCHMARKS"
fi

/usr/local/bin/analyzer \
    --repo="$WORK_DIR" \
    --output=/tmp/metrics.json \
    $BENCHMARK_FLAG

echo ""
echo "=== Uploading to GCS ==="
# Use gcloud to upload (requires workload identity or service account)
# For now, use curl with the GCS JSON API
if command -v gcloud &> /dev/null; then
    gcloud storage cp /tmp/metrics.json "gs://${GCS_BUCKET}/metrics.json"
else
    # Fallback: use gsutil if available
    gsutil cp /tmp/metrics.json "gs://${GCS_BUCKET}/metrics.json"
fi

echo ""
echo "=== Done ==="
echo "Metrics uploaded to gs://${GCS_BUCKET}/metrics.json"

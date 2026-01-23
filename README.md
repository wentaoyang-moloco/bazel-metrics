# Bazel Metrics Dashboard

A React + TypeScript dashboard to visualize Bazel adoption metrics for Go, Python, and Rust monorepos.

## Features

- **Multi-language Support** - Track Go, Python, and Rust packages with language tabs
- **Bazelization Percentage** - % of packages with BUILD files
- **Test Coverage** - % of packages with test files
- **Bazelized Tests** - % of test packages with test targets (go_test, py_test, rust_test)
- **Speed Comparison** - Benchmark `go test` vs `bazel test` (cold/warm cache)
- **Directory Breakdown** - Metrics grouped by top-level directories
- **Package Explorer** - Searchable/filterable table of all packages

## Quick Start

### 1. Run the Analyzer

```bash
cd analyzer
go build -o bazel-metrics ./cmd/main.go
./bazel-metrics --repo=/path/to/your/repo --output=../dashboard/public/metrics.json
```

**Options:**
- `--repo` - Path to repository to analyze (default: `.`)
- `--output` - Output JSON file path (default: `metrics.json`)
- `--benchmark` - Run speed comparison benchmarks
- `--max-benchmarks` - Max packages to benchmark (default: 5)

### 2. Start the Dashboard

```bash
cd dashboard
npm install
npm run dev
```

Open http://localhost:3000

### 3. Deploy to Cloud Run (Optional)

```bash
cd dashboard

# Build and push to Artifact Registry
gcloud builds submit --tag us-central1-docker.pkg.dev/PROJECT_ID/cloud-run-source-deploy/bazel-metrics-dashboard:latest .

# Deploy to Cloud Run
gcloud run deploy bazel-metrics-dashboard \
  --image us-central1-docker.pkg.dev/PROJECT_ID/cloud-run-source-deploy/bazel-metrics-dashboard:latest \
  --region us-central1 \
  --platform managed \
  --allow-unauthenticated \
  --port 8080
```

**Prerequisites:**
- GCP project with Cloud Run, Cloud Build, and Artifact Registry APIs enabled
- `gcloud` CLI authenticated

## Project Structure

```
bazel-metrics/
├── analyzer/                 # Go CLI tool
│   ├── cmd/main.go          # Entry point
│   └── pkg/
│       ├── scanner/         # Scans for BUILD files, packages
│       ├── metrics/         # Calculates percentages
│       └── benchmark/       # Speed comparison runner
├── dashboard/               # React + TypeScript frontend
│   ├── src/
│   │   ├── components/      # UI components
│   │   ├── types/           # TypeScript definitions
│   │   └── App.tsx          # Main dashboard
│   ├── Dockerfile           # Cloud Run container build
│   └── nginx.conf           # Nginx config for serving SPA
└── README.md
```

## Sample Output

```
=== Summary ===

--- Go ---
Packages:        3038
Bazelization:    23.6% (717/3038 packages have BUILD files)
Test Coverage:   62.8% (1907/3038 packages have tests)
Bazelized Tests: 29.0% (packages with tests that have go_test targets)

--- Python ---
Packages:        1190
Bazelization:    11.8% (141/1190 packages have BUILD files)
Test Coverage:   27.4% (326/1190 packages have tests)
Bazelized Tests: 9.8% (packages with tests that have py_test targets)

--- Rust ---
Packages:        25
Bazelization:    32.0% (8/25 packages have BUILD files)
```

## Tech Stack

- **Analyzer**: Go
- **Dashboard**: React, TypeScript, Vite, Tailwind CSS, Recharts

## License

MIT

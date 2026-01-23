package benchmark

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"bazel-metrics/analyzer/pkg/metrics"
	"bazel-metrics/analyzer/pkg/scanner"
)

const commandTimeout = 5 * time.Minute

// Runner executes benchmarks comparing go test vs bazel test
type Runner struct {
	repoPath   string
	scanResult *scanner.ScanResult
	maxTests   int
}

// NewRunner creates a new benchmark runner
func NewRunner(repoPath string, result *scanner.ScanResult, maxTests int) *Runner {
	if maxTests <= 0 {
		maxTests = 5
	}
	return &Runner{
		repoPath:   repoPath,
		scanResult: result,
		maxTests:   maxTests,
	}
}

// Run executes benchmarks and returns speed comparison data
func (r *Runner) Run() (*metrics.SpeedReport, error) {
	report := &metrics.SpeedReport{
		Packages: make([]metrics.PackageBenchmark, 0),
	}

	// Select packages to benchmark (ones with both tests and bazel targets)
	candidates := r.selectCandidates()
	if len(candidates) == 0 {
		return report, nil
	}

	// Limit to maxTests packages
	if len(candidates) > r.maxTests {
		candidates = candidates[:r.maxTests]
	}

	for _, pkg := range candidates {
		benchmark, err := r.benchmarkPackage(pkg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to benchmark %s: %v\n", pkg.RelPath, err)
			continue
		}
		report.Packages = append(report.Packages, *benchmark)
	}

	return report, nil
}

func (r *Runner) selectCandidates() []*scanner.Package {
	var candidates []*scanner.Package

	for _, pkg := range r.scanResult.GoPackages {
		// Package must have test files and go_test targets
		if pkg.HasTestFiles && pkg.TestTargetCount > 0 && pkg.TestFileCount > 0 && pkg.TestFileCount <= 20 {
			candidates = append(candidates, pkg)
		}
	}

	// Sort by test file count (prefer smaller packages for faster benchmarks)
	// Simple selection sort since we only need a few
	for i := 0; i < len(candidates)-1 && i < r.maxTests; i++ {
		minIdx := i
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].TestFileCount < candidates[minIdx].TestFileCount {
				minIdx = j
			}
		}
		candidates[i], candidates[minIdx] = candidates[minIdx], candidates[i]
	}

	return candidates
}

func (r *Runner) benchmarkPackage(pkg *scanner.Package) (*metrics.PackageBenchmark, error) {
	benchmark := &metrics.PackageBenchmark{
		Path: pkg.RelPath,
	}

	// Benchmark go test
	goTestTime, err := r.runGoTest(pkg)
	if err != nil {
		return nil, fmt.Errorf("go test failed: %w", err)
	}
	benchmark.GoTestMs = goTestTime

	// Clean bazel cache for cold run
	r.cleanBazelCache()

	// Benchmark bazel test (cold)
	bazelColdTime, err := r.runBazelTest(pkg)
	if err != nil {
		// Bazel test may fail, but we still want timing
		fmt.Fprintf(os.Stderr, "Warning: bazel test had issues for %s: %v\n", pkg.RelPath, err)
	}
	benchmark.BazelTestColdMs = bazelColdTime

	// Benchmark bazel test (warm - second run)
	bazelWarmTime, err := r.runBazelTest(pkg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: bazel test warm run had issues for %s: %v\n", pkg.RelPath, err)
	}
	benchmark.BazelTestWarmMs = bazelWarmTime

	return benchmark, nil
}

func (r *Runner) runGoTest(pkg *scanner.Package) (int64, error) {
	// Find the go.mod to determine the correct working directory
	goModDir := r.findGoModDir(pkg.Path)
	if goModDir == "" {
		goModDir = r.repoPath
	}

	// Calculate relative path from go.mod directory to package
	relPath, err := filepath.Rel(goModDir, pkg.Path)
	if err != nil {
		relPath = pkg.RelPath
	}
	importPath := "./" + relPath

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "test", "-count=1", importPath)
	cmd.Dir = goModDir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	start := time.Now()
	err = cmd.Run()
	elapsed := time.Since(start).Milliseconds()

	// Tests may fail but we still want timing
	return elapsed, nil
}

func (r *Runner) findGoModDir(pkgPath string) string {
	dir := pkgPath
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir || !strings.HasPrefix(parent, r.repoPath) {
			break
		}
		dir = parent
	}
	return ""
}

func (r *Runner) runBazelTest(pkg *scanner.Package) (int64, error) {
	// Convert path to bazel target
	target := "//" + pkg.RelPath + ":all"

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bazel", "test", target, "--test_output=errors")
	cmd.Dir = r.repoPath

	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start).Milliseconds()

	// Return elapsed time even if test fails
	return elapsed, err
}

func (r *Runner) cleanBazelCache() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Use synchronous clean to ensure cache is fully cleared before benchmark
	cmd := exec.CommandContext(ctx, "bazel", "clean")
	cmd.Dir = r.repoPath
	cmd.Run() // Ignore errors
}

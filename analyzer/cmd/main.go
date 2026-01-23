package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"bazel-metrics/analyzer/pkg/benchmark"
	"bazel-metrics/analyzer/pkg/metrics"
	"bazel-metrics/analyzer/pkg/scanner"
)

func main() {
	var (
		repoPath      string
		outputPath    string
		runBenchmarks bool
		maxBenchmarks int
		prettyPrint   bool
	)

	flag.StringVar(&repoPath, "repo", ".", "Path to the repository to analyze")
	flag.StringVar(&outputPath, "output", "metrics.json", "Output file path for metrics JSON")
	flag.BoolVar(&runBenchmarks, "benchmark", false, "Run speed benchmarks (go test vs bazel test)")
	flag.IntVar(&maxBenchmarks, "max-benchmarks", 5, "Maximum number of packages to benchmark")
	flag.BoolVar(&prettyPrint, "pretty", true, "Pretty print JSON output")
	flag.Parse()

	// Resolve absolute path
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(1)
	}

	// Verify path exists
	if _, err := os.Stat(absRepoPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Repository path does not exist: %s\n", absRepoPath)
		os.Exit(1)
	}

	fmt.Printf("Analyzing repository: %s\n", absRepoPath)

	// Scan repository
	fmt.Println("Scanning for packages and BUILD files...")
	s := scanner.NewScanner(absRepoPath)
	scanResult, err := s.Scan()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Scan error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found: %d Go packages, %d Python packages, %d Rust packages, %d BUILD files\n",
		len(scanResult.GoPackages),
		len(scanResult.PythonPackages),
		len(scanResult.RustPackages),
		scanResult.TotalBUILDs)

	// Calculate metrics
	fmt.Println("Calculating metrics...")
	calc := metrics.NewCalculator(scanResult)
	report := calc.Calculate()

	// Print summary for each language
	fmt.Println("\n=== Summary ===")

	if goSum, ok := report.LanguageSummaries["go"]; ok {
		fmt.Println("\n--- Go ---")
		fmt.Printf("Packages:        %d\n", goSum.TotalPackages)
		fmt.Printf("Bazelization:    %.1f%% (%d/%d packages have BUILD files)\n",
			goSum.BazelizationPct, goSum.PackagesWithBuild, goSum.TotalPackages)
		fmt.Printf("Test Coverage:   %.1f%% (%d/%d packages have tests)\n",
			goSum.TestCoveragePct, goSum.PackagesWithTests, goSum.TotalPackages)
		fmt.Printf("Bazelized Tests: %.1f%% (packages with tests that have go_test targets)\n",
			goSum.BazelizedTestsPct)
		fmt.Printf("Source Files:    %d\n", goSum.TotalSourceFiles)
		fmt.Printf("Test Files:      %d\n", goSum.TotalTestFiles)
		fmt.Printf("Test Targets:    %d\n", goSum.TotalTestTargets)
	}

	if pySum, ok := report.LanguageSummaries["python"]; ok {
		fmt.Println("\n--- Python ---")
		fmt.Printf("Packages:        %d\n", pySum.TotalPackages)
		fmt.Printf("Bazelization:    %.1f%% (%d/%d packages have BUILD files)\n",
			pySum.BazelizationPct, pySum.PackagesWithBuild, pySum.TotalPackages)
		fmt.Printf("Test Coverage:   %.1f%% (%d/%d packages have tests)\n",
			pySum.TestCoveragePct, pySum.PackagesWithTests, pySum.TotalPackages)
		fmt.Printf("Bazelized Tests: %.1f%% (packages with tests that have py_test targets)\n",
			pySum.BazelizedTestsPct)
		fmt.Printf("Source Files:    %d\n", pySum.TotalSourceFiles)
		fmt.Printf("Test Files:      %d\n", pySum.TotalTestFiles)
		fmt.Printf("Test Targets:    %d\n", pySum.TotalTestTargets)
	}

	if rustSum, ok := report.LanguageSummaries["rust"]; ok {
		fmt.Println("\n--- Rust ---")
		fmt.Printf("Packages:        %d\n", rustSum.TotalPackages)
		fmt.Printf("Bazelization:    %.1f%% (%d/%d packages have BUILD files)\n",
			rustSum.BazelizationPct, rustSum.PackagesWithBuild, rustSum.TotalPackages)
		fmt.Printf("Test Coverage:   %.1f%% (%d/%d packages have rust_test targets)\n",
			rustSum.TestCoveragePct, rustSum.PackagesWithTests, rustSum.TotalPackages)
		fmt.Printf("Source Files:    %d\n", rustSum.TotalSourceFiles)
		fmt.Printf("Test Targets:    %d\n", rustSum.TotalTestTargets)
	}

	// Print top directories (Go only)
	if len(report.DirectoryBreakdown) > 0 {
		fmt.Println("\n=== Top Go Directories ===")
		for i, dir := range report.DirectoryBreakdown {
			if i >= 10 {
				break
			}
			fmt.Printf("  %-20s %4d pkgs, %.1f%% bazelized, %.1f%% with tests\n",
				dir.Name, dir.TotalPackages, dir.BazelizationPct, dir.TestCoveragePct)
		}
	}

	// Run benchmarks if requested (Go only for now)
	if runBenchmarks && len(scanResult.GoPackages) > 0 {
		fmt.Println("\n=== Running Speed Benchmarks (Go) ===")
		fmt.Printf("This may take several minutes...\n")

		runner := benchmark.NewRunner(absRepoPath, scanResult, maxBenchmarks)
		speedReport, err := runner.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Benchmark error: %v\n", err)
		} else {
			report.SetSpeedComparison(speedReport)

			fmt.Println("\nBenchmark Results:")
			for _, pkg := range speedReport.Packages {
				fmt.Printf("  %s:\n", pkg.Path)
				fmt.Printf("    go test:          %dms\n", pkg.GoTestMs)
				fmt.Printf("    bazel test (cold): %dms\n", pkg.BazelTestColdMs)
				fmt.Printf("    bazel test (warm): %dms\n", pkg.BazelTestWarmMs)
			}
		}
	}

	// Write output
	fmt.Printf("\nWriting metrics to %s...\n", outputPath)

	var jsonBytes []byte
	if prettyPrint {
		jsonBytes, err = json.MarshalIndent(report, "", "  ")
	} else {
		jsonBytes, err = json.Marshal(report)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON marshal error: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outputPath, jsonBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Write error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done!")
}

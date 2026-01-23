package metrics

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"bazel-metrics/analyzer/pkg/scanner"
)

// LanguageSummary contains metrics for a single language
type LanguageSummary struct {
	Language          string  `json:"language"`
	BazelizationPct   float64 `json:"bazelizationPct"`
	TestCoveragePct   float64 `json:"testCoveragePct"`
	BazelizedTestsPct float64 `json:"bazelizedTestsPct"`
	TotalPackages     int     `json:"totalPackages"`
	TotalSourceFiles  int     `json:"totalSourceFiles"`
	TotalTestFiles    int     `json:"totalTestFiles"`
	PackagesWithBuild int     `json:"packagesWithBuild"`
	PackagesWithTests int     `json:"packagesWithTests"`
	TotalTestTargets  int     `json:"totalTestTargets"`
}

// Summary contains high-level metrics (kept for backwards compatibility)
type Summary struct {
	BazelizationPct    float64 `json:"bazelizationPct"`
	TestCoveragePct    float64 `json:"testCoveragePct"`
	BazelizedTestsPct  float64 `json:"bazelizedTestsPct"`
	TotalPackages      int     `json:"totalPackages"`
	TotalBuildFiles    int     `json:"totalBuildFiles"`
	TotalTestFiles     int     `json:"totalTestFiles"`
	TotalGoFiles       int     `json:"totalGoFiles"`
	PackagesWithBuild  int     `json:"packagesWithBuild"`
	PackagesWithTests  int     `json:"packagesWithTests"`
	TotalGoTestTargets int     `json:"totalGoTestTargets"`
}

// DirectoryMetrics contains metrics grouped by top-level directory
type DirectoryMetrics struct {
	Name              string  `json:"name"`
	TotalPackages     int     `json:"totalPackages"`
	BazelizedPackages int     `json:"bazelizedPackages"`
	PackagesWithTests int     `json:"packagesWithTests"`
	BazelizationPct   float64 `json:"bazelizationPct"`
	TestCoveragePct   float64 `json:"testCoveragePct"`
}

// PackageInfo is the simplified package info for output
type PackageInfo struct {
	Path             string `json:"path"`
	Language         string `json:"language,omitempty"`
	HasBuildFile     bool   `json:"hasBuildFile"`
	HasTestFiles     bool   `json:"hasTestFiles"`
	TestFileCount    int    `json:"testFileCount"`
	TestTargetCount  int    `json:"goTestTargetCount"` // kept as goTestTargetCount for backwards compat
	SourceFileCount  int    `json:"goFileCount"`       // kept as goFileCount for backwards compat
}

// Report is the complete metrics report
type Report struct {
	Timestamp          string              `json:"timestamp"`
	RepoPath           string              `json:"repoPath"`
	Summary            Summary             `json:"summary"`
	DirectoryBreakdown []*DirectoryMetrics `json:"directoryBreakdown"`
	Packages           []*PackageInfo      `json:"packages"`
	SpeedComparison    *SpeedReport        `json:"speedComparison,omitempty"`

	// Multi-language support
	Languages        []string                    `json:"languages"`
	LanguageSummaries map[string]*LanguageSummary `json:"languageSummaries"`
	GoPackages       []*PackageInfo              `json:"goPackages,omitempty"`
	PythonPackages   []*PackageInfo              `json:"pythonPackages,omitempty"`
	RustPackages     []*PackageInfo              `json:"rustPackages,omitempty"`
}

// SpeedReport contains benchmark comparison data
type SpeedReport struct {
	Packages []PackageBenchmark `json:"packages"`
}

// PackageBenchmark contains timing for a single package
type PackageBenchmark struct {
	Path            string `json:"path"`
	GoTestMs        int64  `json:"goTestMs"`
	BazelTestColdMs int64  `json:"bazelTestColdMs"`
	BazelTestWarmMs int64  `json:"bazelTestWarmMs"`
}

// Calculator computes metrics from scan results
type Calculator struct {
	scanResult *scanner.ScanResult
}

// NewCalculator creates a new metrics calculator
func NewCalculator(result *scanner.ScanResult) *Calculator {
	return &Calculator{scanResult: result}
}

// Calculate computes all metrics and returns a report
func (c *Calculator) Calculate() *Report {
	report := &Report{
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
		RepoPath:          c.scanResult.RepoPath,
		Packages:          make([]*PackageInfo, 0),
		Languages:         make([]string, 0),
		LanguageSummaries: make(map[string]*LanguageSummary),
	}

	// Calculate Go metrics
	if len(c.scanResult.GoPackages) > 0 {
		report.Languages = append(report.Languages, "go")
		goSummary := c.calculateLanguageSummary("go", c.scanResult.GoPackages)
		report.LanguageSummaries["go"] = goSummary

		goPackages := make([]*PackageInfo, 0, len(c.scanResult.GoPackages))
		for _, pkg := range c.scanResult.GoPackages {
			pi := &PackageInfo{
				Path:            pkg.RelPath,
				Language:        "go",
				HasBuildFile:    pkg.HasBuildFile,
				HasTestFiles:    pkg.HasTestFiles,
				TestFileCount:   pkg.TestFileCount,
				TestTargetCount: pkg.TestTargetCount,
				SourceFileCount: pkg.SourceFileCount,
			}
			goPackages = append(goPackages, pi)
			report.Packages = append(report.Packages, pi)
		}
		report.GoPackages = goPackages

		// Backwards compatible summary (Go-only)
		report.Summary = Summary{
			BazelizationPct:    goSummary.BazelizationPct,
			TestCoveragePct:    goSummary.TestCoveragePct,
			BazelizedTestsPct:  goSummary.BazelizedTestsPct,
			TotalPackages:      goSummary.TotalPackages,
			TotalBuildFiles:    c.scanResult.TotalBUILDs,
			TotalTestFiles:     goSummary.TotalTestFiles,
			TotalGoFiles:       goSummary.TotalSourceFiles,
			PackagesWithBuild:  goSummary.PackagesWithBuild,
			PackagesWithTests:  goSummary.PackagesWithTests,
			TotalGoTestTargets: goSummary.TotalTestTargets,
		}
	}

	// Calculate Python metrics
	if len(c.scanResult.PythonPackages) > 0 {
		report.Languages = append(report.Languages, "python")
		pySummary := c.calculateLanguageSummary("python", c.scanResult.PythonPackages)
		report.LanguageSummaries["python"] = pySummary

		pyPackages := make([]*PackageInfo, 0, len(c.scanResult.PythonPackages))
		for _, pkg := range c.scanResult.PythonPackages {
			pi := &PackageInfo{
				Path:            pkg.RelPath,
				Language:        "python",
				HasBuildFile:    pkg.HasBuildFile,
				HasTestFiles:    pkg.HasTestFiles,
				TestFileCount:   pkg.TestFileCount,
				TestTargetCount: pkg.TestTargetCount,
				SourceFileCount: pkg.SourceFileCount,
			}
			pyPackages = append(pyPackages, pi)
		}
		report.PythonPackages = pyPackages
	}

	// Calculate Rust metrics
	if len(c.scanResult.RustPackages) > 0 {
		report.Languages = append(report.Languages, "rust")
		rustSummary := c.calculateLanguageSummary("rust", c.scanResult.RustPackages)
		report.LanguageSummaries["rust"] = rustSummary

		rustPackages := make([]*PackageInfo, 0, len(c.scanResult.RustPackages))
		for _, pkg := range c.scanResult.RustPackages {
			pi := &PackageInfo{
				Path:            pkg.RelPath,
				Language:        "rust",
				HasBuildFile:    pkg.HasBuildFile,
				HasTestFiles:    pkg.HasTestFiles,
				TestFileCount:   pkg.TestFileCount,
				TestTargetCount: pkg.TestTargetCount,
				SourceFileCount: pkg.SourceFileCount,
			}
			rustPackages = append(rustPackages, pi)
		}
		report.RustPackages = rustPackages
	}

	// Calculate directory breakdown (Go only for backwards compat)
	report.DirectoryBreakdown = c.calculateDirectoryBreakdown(c.scanResult.GoPackages)

	return report
}

func (c *Calculator) calculateLanguageSummary(lang string, packages []*scanner.Package) *LanguageSummary {
	summary := &LanguageSummary{
		Language:      lang,
		TotalPackages: len(packages),
	}

	for _, pkg := range packages {
		summary.TotalSourceFiles += pkg.SourceFileCount
		summary.TotalTestFiles += pkg.TestFileCount
		summary.TotalTestTargets += pkg.TestTargetCount

		if pkg.HasBuildFile {
			summary.PackagesWithBuild++
		}
		if pkg.HasTestFiles {
			summary.PackagesWithTests++
		}
	}

	// Calculate percentages
	if summary.TotalPackages > 0 {
		summary.BazelizationPct = float64(summary.PackagesWithBuild) / float64(summary.TotalPackages) * 100
		summary.TestCoveragePct = float64(summary.PackagesWithTests) / float64(summary.TotalPackages) * 100
	}

	// Bazelized tests: packages with tests that also have test targets
	packagesWithBazelizedTests := 0
	for _, pkg := range packages {
		if pkg.HasTestFiles && pkg.TestTargetCount > 0 {
			packagesWithBazelizedTests++
		}
	}
	if summary.PackagesWithTests > 0 {
		summary.BazelizedTestsPct = float64(packagesWithBazelizedTests) / float64(summary.PackagesWithTests) * 100
	}

	return summary
}

func (c *Calculator) calculateDirectoryBreakdown(packages []*scanner.Package) []*DirectoryMetrics {
	dirMap := make(map[string]*DirectoryMetrics)

	for _, pkg := range packages {
		// Get top-level directory (first component of path)
		topDir := getTopLevelDir(pkg.RelPath)
		if topDir == "" || topDir == "." {
			topDir = "(root)"
		}

		dm, exists := dirMap[topDir]
		if !exists {
			dm = &DirectoryMetrics{Name: topDir}
			dirMap[topDir] = dm
		}

		dm.TotalPackages++
		if pkg.HasBuildFile {
			dm.BazelizedPackages++
		}
		if pkg.HasTestFiles {
			dm.PackagesWithTests++
		}
	}

	// Calculate percentages and convert to slice
	result := make([]*DirectoryMetrics, 0, len(dirMap))
	for _, dm := range dirMap {
		if dm.TotalPackages > 0 {
			dm.BazelizationPct = float64(dm.BazelizedPackages) / float64(dm.TotalPackages) * 100
			dm.TestCoveragePct = float64(dm.PackagesWithTests) / float64(dm.TotalPackages) * 100
		}
		result = append(result, dm)
	}

	// Sort by total packages descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalPackages > result[j].TotalPackages
	})

	return result
}

func getTopLevelDir(path string) string {
	// Clean the path and get the first component
	path = filepath.Clean(path)
	parts := strings.Split(path, string(filepath.Separator))
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// SetSpeedComparison adds speed comparison data to the report
func (r *Report) SetSpeedComparison(speed *SpeedReport) {
	r.SpeedComparison = speed
}

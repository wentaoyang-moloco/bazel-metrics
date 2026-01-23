package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Language represents a programming language
type Language string

const (
	LangGo     Language = "go"
	LangPython Language = "python"
	LangRust   Language = "rust"
)

// Package represents a package directory with its metadata
type Package struct {
	Path             string   `json:"path"`
	RelPath          string   `json:"relPath"`
	Language         Language `json:"language"`
	HasBuildFile     bool     `json:"hasBuildFile"`
	HasTestFiles     bool     `json:"hasTestFiles"`
	SourceFileCount  int      `json:"sourceFileCount"`
	TestFileCount    int      `json:"testFileCount"`
	TestTargetCount  int      `json:"testTargetCount"`
	LibraryTargets   int      `json:"libraryTargetCount"`
	BinaryTargets    int      `json:"binaryTargetCount"`
}

// ScanResult contains the complete scan results
type ScanResult struct {
	RepoPath string `json:"repoPath"`

	// Per-language packages
	GoPackages     []*Package `json:"goPackages"`
	PythonPackages []*Package `json:"pythonPackages"`
	RustPackages   []*Package `json:"rustPackages"`

	// Totals
	TotalBUILDs int `json:"totalBuildFiles"`

	// Go totals
	TotalGoFiles     int `json:"totalGoFiles"`
	TotalGoTests     int `json:"totalGoTestFiles"`
	TotalGoTestRules int `json:"totalGoTestRules"`

	// Python totals
	TotalPythonFiles     int `json:"totalPythonFiles"`
	TotalPythonTests     int `json:"totalPythonTestFiles"`
	TotalPyTestRules     int `json:"totalPyTestRules"`

	// Rust totals
	TotalRustFiles     int `json:"totalRustFiles"`
	TotalRustTests     int `json:"totalRustTestFiles"`
	TotalRustTestRules int `json:"totalRustTestRules"`
}

// Scanner scans a repository for Bazel and language metrics
type Scanner struct {
	repoPath string
	skipDirs map[string]bool

	// Go regex patterns
	goTestRegex *regexp.Regexp
	goLibRegex  *regexp.Regexp
	goBinRegex  *regexp.Regexp

	// Python regex patterns
	pyTestRegex *regexp.Regexp
	pyLibRegex  *regexp.Regexp
	pyBinRegex  *regexp.Regexp

	// Rust regex patterns
	rustTestRegex *regexp.Regexp
	rustLibRegex  *regexp.Regexp
	rustBinRegex  *regexp.Regexp
}

// NewScanner creates a new scanner for the given repository path
func NewScanner(repoPath string) *Scanner {
	return &Scanner{
		repoPath: repoPath,
		skipDirs: map[string]bool{
			".git":           true,
			"bazel-bin":      true,
			"bazel-out":      true,
			"bazel-testlogs": true,
			"node_modules":   true,
			".cache":         true,
			"vendor":         true,
			"__pycache__":    true,
			"target":         true, // Rust target directory
			".venv":          true,
			"venv":           true,
		},
		// Go patterns
		goTestRegex: regexp.MustCompile(`(?m)^\s*go_test\s*\(`),
		goLibRegex:  regexp.MustCompile(`(?m)^\s*go_library\s*\(`),
		goBinRegex:  regexp.MustCompile(`(?m)^\s*go_binary\s*\(`),
		// Python patterns
		pyTestRegex: regexp.MustCompile(`(?m)^\s*py_test\s*\(`),
		pyLibRegex:  regexp.MustCompile(`(?m)^\s*py_library\s*\(`),
		pyBinRegex:  regexp.MustCompile(`(?m)^\s*py_binary\s*\(`),
		// Rust patterns
		rustTestRegex: regexp.MustCompile(`(?m)^\s*rust_test\s*\(`),
		rustLibRegex:  regexp.MustCompile(`(?m)^\s*rust_library\s*\(`),
		rustBinRegex:  regexp.MustCompile(`(?m)^\s*rust_binary\s*\(`),
	}
}

// dirPackages holds package info for a single directory, per language
type dirPackages struct {
	path        string
	relPath     string
	hasBuild    bool
	targets     *buildTargets
	goPkg       *Package
	pythonPkg   *Package
	rustPkg     *Package
}

// Scan performs a full scan of the repository
func (s *Scanner) Scan() (*ScanResult, error) {
	result := &ScanResult{
		RepoPath:       s.repoPath,
		GoPackages:     make([]*Package, 0),
		PythonPackages: make([]*Package, 0),
		RustPackages:   make([]*Package, 0),
	}

	dirMap := make(map[string]*dirPackages)

	err := filepath.Walk(s.repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip hidden and excluded directories
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || s.skipDirs[base] || strings.HasPrefix(base, "bazel-") {
				return filepath.SkipDir
			}
			return nil
		}

		dir := filepath.Dir(path)
		relDir, err := filepath.Rel(s.repoPath, dir)
		if err != nil || relDir == "" {
			relDir = "."
		}

		// Get or create dir entry
		dp, exists := dirMap[dir]
		if !exists {
			dp = &dirPackages{
				path:    dir,
				relPath: relDir,
			}
			dirMap[dir] = dp
		}

		filename := filepath.Base(path)

		// Check for BUILD files
		if filename == "BUILD" || filename == "BUILD.bazel" {
			dp.hasBuild = true
			result.TotalBUILDs++

			// Parse BUILD file for targets
			targets, err := s.parseBuildFile(path)
			if err == nil {
				dp.targets = targets
			}
		}

		// Check for Go files
		if strings.HasSuffix(filename, ".go") {
			if dp.goPkg == nil {
				dp.goPkg = &Package{
					Path:     dir,
					RelPath:  relDir,
					Language: LangGo,
				}
			}
			if strings.HasSuffix(filename, "_test.go") {
				dp.goPkg.HasTestFiles = true
				dp.goPkg.TestFileCount++
				result.TotalGoTests++
			} else {
				dp.goPkg.SourceFileCount++
				result.TotalGoFiles++
			}
		}

		// Check for Python files
		if strings.HasSuffix(filename, ".py") {
			if dp.pythonPkg == nil {
				dp.pythonPkg = &Package{
					Path:     dir,
					RelPath:  relDir,
					Language: LangPython,
				}
			}
			// Python test patterns: *_test.py, test_*.py, *_tests.py
			if strings.HasSuffix(filename, "_test.py") ||
				strings.HasPrefix(filename, "test_") ||
				strings.HasSuffix(filename, "_tests.py") {
				dp.pythonPkg.HasTestFiles = true
				dp.pythonPkg.TestFileCount++
				result.TotalPythonTests++
			} else {
				dp.pythonPkg.SourceFileCount++
				result.TotalPythonFiles++
			}
		}

		// Check for Rust files
		if strings.HasSuffix(filename, ".rs") {
			if dp.rustPkg == nil {
				dp.rustPkg = &Package{
					Path:     dir,
					RelPath:  relDir,
					Language: LangRust,
				}
			}
			// Rust doesn't have separate test files - tests are usually inline
			// We'll count all .rs files as source files
			dp.rustPkg.SourceFileCount++
			result.TotalRustFiles++
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Process all directories and assign BUILD targets
	for _, dp := range dirMap {
		// Assign BUILD file info and targets to packages
		if dp.goPkg != nil {
			dp.goPkg.HasBuildFile = dp.hasBuild
			if dp.targets != nil {
				dp.goPkg.TestTargetCount = dp.targets.goTests
				dp.goPkg.LibraryTargets = dp.targets.goLibs
				dp.goPkg.BinaryTargets = dp.targets.goBins
				result.TotalGoTestRules += dp.targets.goTests
			}
			result.GoPackages = append(result.GoPackages, dp.goPkg)
		}

		if dp.pythonPkg != nil {
			dp.pythonPkg.HasBuildFile = dp.hasBuild
			if dp.targets != nil {
				dp.pythonPkg.TestTargetCount = dp.targets.pyTests
				dp.pythonPkg.LibraryTargets = dp.targets.pyLibs
				dp.pythonPkg.BinaryTargets = dp.targets.pyBins
				result.TotalPyTestRules += dp.targets.pyTests
			}
			result.PythonPackages = append(result.PythonPackages, dp.pythonPkg)
		}

		if dp.rustPkg != nil {
			dp.rustPkg.HasBuildFile = dp.hasBuild
			if dp.targets != nil {
				dp.rustPkg.TestTargetCount = dp.targets.rustTests
				dp.rustPkg.LibraryTargets = dp.targets.rustLibs
				dp.rustPkg.BinaryTargets = dp.targets.rustBins
				result.TotalRustTestRules += dp.targets.rustTests
				// For Rust, if there are rust_test targets, mark as having tests
				if dp.targets.rustTests > 0 {
					dp.rustPkg.HasTestFiles = true
					dp.rustPkg.TestFileCount = dp.targets.rustTests
					result.TotalRustTests += dp.targets.rustTests
				}
			}
			result.RustPackages = append(result.RustPackages, dp.rustPkg)
		}
	}

	// Sort packages by path for deterministic output
	sort.Slice(result.GoPackages, func(i, j int) bool {
		return result.GoPackages[i].RelPath < result.GoPackages[j].RelPath
	})
	sort.Slice(result.PythonPackages, func(i, j int) bool {
		return result.PythonPackages[i].RelPath < result.PythonPackages[j].RelPath
	})
	sort.Slice(result.RustPackages, func(i, j int) bool {
		return result.RustPackages[i].RelPath < result.RustPackages[j].RelPath
	})

	return result, nil
}

type buildTargets struct {
	// Go targets
	goTests int
	goLibs  int
	goBins  int
	// Python targets
	pyTests int
	pyLibs  int
	pyBins  int
	// Rust targets
	rustTests int
	rustLibs  int
	rustBins  int
}

func (s *Scanner) parseBuildFile(path string) (*buildTargets, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	targets := &buildTargets{}
	sc := bufio.NewScanner(file)
	var content strings.Builder

	for sc.Scan() {
		content.WriteString(sc.Text())
		content.WriteString("\n")
	}

	text := content.String()

	// Go targets
	targets.goTests = len(s.goTestRegex.FindAllString(text, -1))
	targets.goLibs = len(s.goLibRegex.FindAllString(text, -1))
	targets.goBins = len(s.goBinRegex.FindAllString(text, -1))

	// Python targets
	targets.pyTests = len(s.pyTestRegex.FindAllString(text, -1))
	targets.pyLibs = len(s.pyLibRegex.FindAllString(text, -1))
	targets.pyBins = len(s.pyBinRegex.FindAllString(text, -1))

	// Rust targets
	targets.rustTests = len(s.rustTestRegex.FindAllString(text, -1))
	targets.rustLibs = len(s.rustLibRegex.FindAllString(text, -1))
	targets.rustBins = len(s.rustBinRegex.FindAllString(text, -1))

	return targets, sc.Err()
}

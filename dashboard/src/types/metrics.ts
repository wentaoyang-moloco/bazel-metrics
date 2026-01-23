export type Language = 'go' | 'python' | 'rust';

export interface LanguageSummary {
  language: string;
  bazelizationPct: number;
  testCoveragePct: number;
  bazelizedTestsPct: number;
  totalPackages: number;
  totalSourceFiles: number;
  totalTestFiles: number;
  packagesWithBuild: number;
  packagesWithTests: number;
  totalTestTargets: number;
}

// Kept for backwards compatibility
export interface Summary {
  bazelizationPct: number;
  testCoveragePct: number;
  bazelizedTestsPct: number;
  totalPackages: number;
  totalBuildFiles: number;
  totalTestFiles: number;
  totalGoFiles: number;
  packagesWithBuild: number;
  packagesWithTests: number;
  totalGoTestTargets: number;
}

export interface DirectoryMetrics {
  name: string;
  totalPackages: number;
  bazelizedPackages: number;
  packagesWithTests: number;
  bazelizationPct: number;
  testCoveragePct: number;
}

export interface PackageInfo {
  path: string;
  language?: string;
  hasBuildFile: boolean;
  hasTestFiles: boolean;
  testFileCount: number;
  goTestTargetCount: number;  // kept for backwards compat, represents testTargetCount
  goFileCount: number;        // kept for backwards compat, represents sourceFileCount
}

export interface PackageBenchmark {
  path: string;
  goTestMs: number;
  bazelTestColdMs: number;
  bazelTestWarmMs: number;
}

export interface SpeedReport {
  packages: PackageBenchmark[];
}

export interface MetricsReport {
  timestamp: string;
  repoPath: string;
  summary: Summary;
  directoryBreakdown: DirectoryMetrics[];
  packages: PackageInfo[];
  speedComparison?: SpeedReport;

  // Multi-language support
  languages?: string[];
  languageSummaries?: Record<string, LanguageSummary>;
  goPackages?: PackageInfo[];
  pythonPackages?: PackageInfo[];
  rustPackages?: PackageInfo[];
}

import { useState, useMemo } from 'react';
import type { PackageInfo, PackageBenchmark } from '../types/metrics';

type Language = 'go' | 'python' | 'rust';

interface PackageExplorerProps {
  packages: PackageInfo[];
  benchmarks?: PackageBenchmark[];
  language?: Language;
}

type FilterOption = 'all' | 'bazelized' | 'not-bazelized' | 'with-tests' | 'without-tests';

const languageConfig: Record<Language, { testLabel: string; sourceLabel: string; fileExt: string }> = {
  go: { testLabel: 'Bazel Tests', sourceLabel: 'Source Go Files', fileExt: '_test.go' },
  python: { testLabel: 'py_test', sourceLabel: 'Source Py Files', fileExt: '_test.py' },
  rust: { testLabel: 'rust_test', sourceLabel: 'Rust Files', fileExt: '.rs' },
};

export function PackageExplorer({ packages, benchmarks = [], language = 'go' }: PackageExplorerProps) {
  const [filter, setFilter] = useState<FilterOption>('all');
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(0);
  const pageSize = 20;

  const config = languageConfig[language];

  // Reset page when language/packages change
  const packagesKey = `${language}-${packages.length}`;
  const [prevPackagesKey, setPrevPackagesKey] = useState(packagesKey);
  if (packagesKey !== prevPackagesKey) {
    setPage(0);
    setPrevPackagesKey(packagesKey);
  }

  // Create a map of path -> benchmark data for quick lookup
  const benchmarkMap = useMemo(() => {
    const map = new Map<string, PackageBenchmark>();
    benchmarks.forEach(b => map.set(b.path, b));
    return map;
  }, [benchmarks]);

  const formatMs = (ms: number | undefined) => {
    if (ms === undefined) return null;
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(1)}s`;
  };

  const filteredPackages = useMemo(() => {
    const filtered = packages.filter(pkg => {
      // Search filter
      if (search && !pkg.path.toLowerCase().includes(search.toLowerCase())) {
        return false;
      }

      // Category filter
      switch (filter) {
        case 'bazelized':
          return pkg.hasBuildFile;
        case 'not-bazelized':
          return !pkg.hasBuildFile;
        case 'with-tests':
          return pkg.hasTestFiles;
        case 'without-tests':
          return !pkg.hasTestFiles;
        default:
          return true;
      }
    });

    // Sort priority:
    // 0. BUILD + test files + test targets (fully bazelized with local tests)
    // 1. BUILD + test targets only (targets may reference external tests)
    // 2. BUILD + test files but no test targets (unbazelized tests)
    // 3. BUILD only (no tests)
    // 4. No BUILD
    // Within each priority, sort alphabetically
    return [...filtered].sort((a, b) => {
      const getSortPriority = (pkg: PackageInfo) => {
        // goTestTargetCount is used for all languages (go_test, py_test, rust_test)
        if (pkg.hasBuildFile && pkg.goTestTargetCount > 0 && pkg.hasTestFiles) return 0; // Fully bazelized with local test files
        if (pkg.hasBuildFile && pkg.goTestTargetCount > 0) return 1; // Has test targets but no local test files
        if (pkg.hasBuildFile && pkg.hasTestFiles) return 2; // BUILD + test files but no test targets
        if (pkg.hasBuildFile) return 3; // BUILD only
        return 4; // No BUILD
      };

      const aPriority = getSortPriority(a);
      const bPriority = getSortPriority(b);

      if (aPriority !== bPriority) return aPriority - bPriority;

      // Within same priority, sort alphabetically
      return a.path.localeCompare(b.path);
    });
  }, [packages, filter, search]);

  const paginatedPackages = filteredPackages.slice(page * pageSize, (page + 1) * pageSize);
  const totalPages = Math.ceil(filteredPackages.length / pageSize);

  // Check if we should show benchmark columns (only for Go with benchmarks)
  const showBenchmarks = language === 'go' && benchmarks.length > 0;

  return (
    <div className="metric-card">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center mb-4 gap-4">
        <h3 className="text-lg font-semibold">Package Explorer</h3>
        <div className="flex flex-wrap gap-2">
          <input
            type="text"
            placeholder="Search packages..."
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(0); }}
            className="px-3 py-1.5 bg-bb-accent/50 border border-bb-accent rounded text-sm text-white placeholder-gray-400"
          />
          <select
            value={filter}
            onChange={(e) => { setFilter(e.target.value as FilterOption); setPage(0); }}
            className="px-3 py-1.5 bg-bb-accent/50 border border-bb-accent rounded text-sm text-white"
          >
            <option value="all">All Packages</option>
            <option value="bazelized">Bazelized Only</option>
            <option value="not-bazelized">Not Bazelized</option>
            <option value="with-tests">With Tests</option>
            <option value="without-tests">Without Tests</option>
          </select>
        </div>
      </div>

      <div className="text-sm text-gray-400 mb-2">
        Showing {paginatedPackages.length} of {filteredPackages.length} packages
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="text-left text-gray-400 border-b border-bb-accent">
              <th className="pb-2 pr-4">Path</th>
              <th className="pb-2 pr-4 text-center">BUILD</th>
              <th className="pb-2 pr-4 text-center" title={`Number of ${config.fileExt} files`}>Test Files</th>
              <th className="pb-2 pr-4 text-center" title={`Number of ${config.testLabel} targets in BUILD file`}>{config.testLabel}</th>
              <th className="pb-2 pr-4 text-center" title="Non-test source files">{config.sourceLabel}</th>
              {showBenchmarks && (
                <>
                  <th className="pb-2 pr-4 text-center" title="go test execution time">Go Test Time</th>
                  <th className="pb-2 pr-4 text-center" title="bazel test execution time (cold cache)">Bazel Cold</th>
                  <th className="pb-2 text-center" title="bazel test execution time (warm cache)">Bazel Warm</th>
                </>
              )}
            </tr>
          </thead>
          <tbody>
            {paginatedPackages.map((pkg) => (
              <tr key={pkg.path} className="table-row border-b border-bb-accent/30">
                <td className="py-2 pr-4 font-mono text-xs">{pkg.path}</td>
                <td className="py-2 pr-4 text-center">
                  {pkg.hasBuildFile ? (
                    <span className="text-green-400">✓</span>
                  ) : (
                    <span className="text-gray-600">-</span>
                  )}
                </td>
                <td className="py-2 pr-4 text-center">
                  {pkg.hasTestFiles ? (
                    <span className="text-blue-400">{pkg.testFileCount}</span>
                  ) : (
                    <span className="text-gray-600">-</span>
                  )}
                </td>
                <td className="py-2 pr-4 text-center">
                  {pkg.goTestTargetCount > 0 ? (
                    <span className="text-green-400">{pkg.goTestTargetCount}</span>
                  ) : (
                    <span className="text-gray-600">-</span>
                  )}
                </td>
                <td className="py-2 pr-4 text-center text-gray-400">{pkg.goFileCount}</td>
                {showBenchmarks && (
                  <>
                    <td className="py-2 pr-4 text-center">
                      {benchmarkMap.get(pkg.path)?.goTestMs !== undefined ? (
                        <span className="text-blue-400">{formatMs(benchmarkMap.get(pkg.path)?.goTestMs)}</span>
                      ) : (
                        <span className="text-gray-600">-</span>
                      )}
                    </td>
                    <td className="py-2 pr-4 text-center">
                      {benchmarkMap.get(pkg.path)?.bazelTestColdMs !== undefined ? (
                        <span className="text-orange-400">{formatMs(benchmarkMap.get(pkg.path)?.bazelTestColdMs)}</span>
                      ) : (
                        <span className="text-gray-600">-</span>
                      )}
                    </td>
                    <td className="py-2 text-center">
                      {benchmarkMap.get(pkg.path)?.bazelTestWarmMs !== undefined ? (
                        <span className="text-green-400">{formatMs(benchmarkMap.get(pkg.path)?.bazelTestWarmMs)}</span>
                      ) : (
                        <span className="text-gray-600">-</span>
                      )}
                    </td>
                  </>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div className="flex justify-center gap-2 mt-4">
          <button
            onClick={() => setPage(p => Math.max(0, p - 1))}
            disabled={page === 0}
            className="px-3 py-1 bg-bb-accent/50 rounded disabled:opacity-50"
          >
            Prev
          </button>
          <span className="px-3 py-1 text-gray-400">
            Page {page + 1} of {totalPages}
          </span>
          <button
            onClick={() => setPage(p => Math.min(totalPages - 1, p + 1))}
            disabled={page >= totalPages - 1}
            className="px-3 py-1 bg-bb-accent/50 rounded disabled:opacity-50"
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
}

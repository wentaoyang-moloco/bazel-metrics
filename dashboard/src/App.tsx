import { useState, useEffect } from 'react';
import type { MetricsReport, LanguageSummary, PackageInfo } from './types/metrics';
import { MetricCard } from './components/MetricCard';
import { GaugeCircle } from './components/GaugeCircle';
import { DirectoryBreakdown } from './components/DirectoryBreakdown';
import { PackageExplorer } from './components/PackageExplorer';
import { SpeedComparison } from './components/SpeedComparison';

type Language = 'go' | 'python' | 'rust';

const languageLabels: Record<Language, string> = {
  go: 'Go',
  python: 'Python',
  rust: 'Rust',
};


function App() {
  const [metrics, setMetrics] = useState<MetricsReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeLanguage, setActiveLanguage] = useState<Language>('go');

  useEffect(() => {
    fetch('/metrics.json')
      .then(res => {
        if (!res.ok) throw new Error('Failed to load metrics.json');
        return res.json();
      })
      .then(data => {
        setMetrics(data);
        setLoading(false);
        // Set default language to first available
        if (data.languages && data.languages.length > 0) {
          setActiveLanguage(data.languages[0] as Language);
        }
      })
      .catch(err => {
        setError(err.message);
        setLoading(false);
      });
  }, []);

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-xl text-gray-400">Loading metrics...</div>
      </div>
    );
  }

  if (error || !metrics) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl text-red-400 mb-4">Error Loading Metrics</h1>
          <p className="text-gray-400">{error || 'No metrics data available'}</p>
          <p className="text-gray-500 mt-4">
            Run the analyzer first: <code className="bg-bb-accent px-2 py-1 rounded">npm run analyze</code>
          </p>
        </div>
      </div>
    );
  }

  const repoName = metrics.repoPath.split('/').pop() || 'Repository';
  const timestamp = new Date(metrics.timestamp).toLocaleString();

  // Get available languages
  const availableLanguages = (metrics.languages || ['go']) as Language[];

  // Get current language summary
  const currentSummary: LanguageSummary | null = metrics.languageSummaries?.[activeLanguage] || null;

  // Get packages for current language
  const getPackagesForLanguage = (lang: Language): PackageInfo[] => {
    switch (lang) {
      case 'go':
        return metrics.goPackages || metrics.packages;
      case 'python':
        return metrics.pythonPackages || [];
      case 'rust':
        return metrics.rustPackages || [];
      default:
        return [];
    }
  };

  const currentPackages = getPackagesForLanguage(activeLanguage);

  // For backwards compatibility, use summary if no languageSummaries
  const displaySummary = currentSummary || {
    language: 'go',
    bazelizationPct: metrics.summary.bazelizationPct,
    testCoveragePct: metrics.summary.testCoveragePct,
    bazelizedTestsPct: metrics.summary.bazelizedTestsPct,
    totalPackages: metrics.summary.totalPackages,
    totalSourceFiles: metrics.summary.totalGoFiles,
    totalTestFiles: metrics.summary.totalTestFiles,
    packagesWithBuild: metrics.summary.packagesWithBuild,
    packagesWithTests: metrics.summary.packagesWithTests,
    totalTestTargets: metrics.summary.totalGoTestTargets,
  };

  // Labels based on language
  const testTargetLabel = activeLanguage === 'go' ? 'go_test' :
                          activeLanguage === 'python' ? 'py_test' : 'rust_test';
  const fileExtension = activeLanguage === 'go' ? '.go' :
                        activeLanguage === 'python' ? '.py' : '.rs';

  return (
    <div className="min-h-screen p-6">
      {/* Header */}
      <header className="mb-8">
        <h1 className="text-3xl font-bold text-white mb-2">Bazel Metrics Dashboard</h1>
        <div className="flex flex-wrap gap-4 text-sm text-gray-400">
          <span>Repository: <code className="text-blue-400">{repoName}</code></span>
          <span>Last scan: {timestamp}</span>
        </div>
      </header>

      {/* Language Tabs */}
      {availableLanguages.length > 1 && (
        <div className="mb-6">
          <div className="flex gap-2 border-b border-bb-accent">
            {availableLanguages.map(lang => (
              <button
                key={lang}
                onClick={() => setActiveLanguage(lang)}
                className={`px-4 py-2 text-sm font-medium transition-colors ${
                  activeLanguage === lang
                    ? 'text-white border-b-2 border-blue-500 -mb-px'
                    : 'text-gray-400 hover:text-white'
                }`}
              >
                {languageLabels[lang]}
                <span className="ml-2 text-xs text-gray-500">
                  ({metrics.languageSummaries?.[lang]?.totalPackages || 0} packages)
                </span>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Gauges Row */}
      <section className="metric-card mb-6">
        <div className="flex flex-wrap justify-around gap-8">
          <div className="relative">
            <GaugeCircle
              percentage={displaySummary.bazelizationPct}
              label="Bazelization"
              sublabel={`${displaySummary.packagesWithBuild}/${displaySummary.totalPackages} packages`}
              size={140}
            />
          </div>
          <div className="relative">
            <GaugeCircle
              percentage={displaySummary.testCoveragePct}
              label="Test Coverage"
              sublabel={`${displaySummary.packagesWithTests}/${displaySummary.totalPackages} packages`}
              size={140}
            />
          </div>
          <div className="relative">
            <GaugeCircle
              percentage={displaySummary.bazelizedTestsPct}
              label="Bazelized Tests"
              sublabel={`packages with ${testTargetLabel} targets`}
              size={140}
            />
          </div>
        </div>
      </section>

      {/* Summary Cards */}
      <section className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <MetricCard
          title="Total Packages"
          value={displaySummary.totalPackages.toLocaleString()}
          subtitle={`${languageLabels[activeLanguage]} packages found`}
          color="blue"
        />
        <MetricCard
          title="BUILD Files"
          value={displaySummary.packagesWithBuild.toLocaleString()}
          subtitle="Packages with BUILD"
          color="green"
        />
        <MetricCard
          title="Test Files"
          value={displaySummary.totalTestFiles.toLocaleString()}
          subtitle={`*_test${fileExtension} files`}
          color="yellow"
        />
        <MetricCard
          title={`${testTargetLabel} Targets`}
          value={displaySummary.totalTestTargets.toLocaleString()}
          subtitle="Bazel test targets"
          color="green"
        />
      </section>

      {/* Directory Breakdown and Speed Comparison - Only for Go */}
      {activeLanguage === 'go' && (
        <section className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          <DirectoryBreakdown data={metrics.directoryBreakdown} />
          <SpeedComparison data={metrics.speedComparison} />
        </section>
      )}

      {/* Package Explorer */}
      <section>
        <PackageExplorer
          packages={currentPackages}
          benchmarks={activeLanguage === 'go' ? metrics.speedComparison?.packages : undefined}
          language={activeLanguage}
        />
      </section>

    </div>
  );
}

export default App;

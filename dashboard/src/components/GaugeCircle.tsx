interface GaugeCircleProps {
  percentage: number;
  label: string;
  sublabel?: string;
  size?: number;
}

export function GaugeCircle({ percentage, label, sublabel, size = 120 }: GaugeCircleProps) {
  // Clamp percentage to valid range
  const clampedPct = Math.max(0, Math.min(100, Number.isFinite(percentage) ? percentage : 0));

  const strokeWidth = 8;
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const strokeDashoffset = circumference - (clampedPct / 100) * circumference;

  const getColor = (pct: number) => {
    if (pct >= 75) return '#22c55e'; // green
    if (pct >= 50) return '#f59e0b'; // yellow
    if (pct >= 25) return '#f97316'; // orange
    return '#ef4444'; // red
  };

  return (
    <div className="relative flex flex-col items-center">
      <svg width={size} height={size} className="transform -rotate-90">
        {/* Background circle */}
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          stroke="#374151"
          strokeWidth={strokeWidth}
          fill="none"
        />
        {/* Progress circle */}
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          stroke={getColor(clampedPct)}
          strokeWidth={strokeWidth}
          fill="none"
          strokeDasharray={circumference}
          strokeDashoffset={strokeDashoffset}
          strokeLinecap="round"
          className="gauge-circle"
        />
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <span className="text-2xl font-bold text-white">{clampedPct.toFixed(1)}%</span>
      </div>
      <div className="mt-2 text-center">
        <p className="text-sm font-medium text-gray-300">{label}</p>
        {sublabel && <p className="text-xs text-gray-500">{sublabel}</p>}
      </div>
    </div>
  );
}

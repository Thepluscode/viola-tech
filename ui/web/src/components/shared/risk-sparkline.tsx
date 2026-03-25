"use client";

/**
 * RiskSparkline — lightweight inline SVG sparkline for risk score trends.
 * Renders a 60×20 px path from an array of 0-100 values.
 * No external chart library required — pure SVG.
 */
interface RiskSparklineProps {
  values: number[];        // Array of 0-100 risk scores, oldest first
  width?: number;
  height?: number;
  color?: string;          // Tailwind-safe hex or CSS colour
  className?: string;
}

export function RiskSparkline({
  values,
  width = 64,
  height = 24,
  color = "#00d4ff",
  className,
}: RiskSparklineProps) {
  if (values.length < 2) return null;

  const pad = 2;
  const w = width - pad * 2;
  const h = height - pad * 2;

  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;

  const points = values.map((v, i) => {
    const x = pad + (i / (values.length - 1)) * w;
    const y = pad + h - ((v - min) / range) * h;
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  });

  const pathD = `M ${points.join(" L ")}`;
  const lastVal = values[values.length - 1];

  // Fill area under the sparkline
  const fillD =
    `M ${points[0]} L ${points.join(" L ")} ` +
    `L ${(pad + w).toFixed(1)},${(pad + h).toFixed(1)} ` +
    `L ${pad},${(pad + h).toFixed(1)} Z`;

  return (
    <svg
      width={width}
      height={height}
      viewBox={`0 0 ${width} ${height}`}
      className={className}
      aria-label={`Risk trend, latest: ${lastVal.toFixed(0)}`}
      role="img"
    >
      <defs>
        <linearGradient id="spark-fill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={color} stopOpacity="0.3" />
          <stop offset="100%" stopColor={color} stopOpacity="0.02" />
        </linearGradient>
      </defs>
      <path d={fillD} fill="url(#spark-fill)" />
      <path
        d={pathD}
        fill="none"
        stroke={color}
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

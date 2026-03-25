import { cn } from "@/lib/utils";

interface RiskScoreBarProps {
  score: number;
  showLabel?: boolean;
  className?: string;
}

function getRiskColor(score: number): string {
  if (score >= 80) return "bg-severity-critical";
  if (score >= 60) return "bg-severity-high";
  if (score >= 40) return "bg-severity-medium";
  return "bg-severity-low";
}

export function RiskScoreBar({ score, showLabel = true, className }: RiskScoreBarProps) {
  const color = getRiskColor(score);
  const clamped = Math.min(100, Math.max(0, score));

  return (
    <div className={cn("flex items-center gap-2", className)}>
      {showLabel && (
        <span className="text-xs font-mono text-viola-text w-6 text-right">{clamped}</span>
      )}
      <div className="flex-1 h-1.5 bg-viola-border rounded-md overflow-hidden min-w-[60px]">
        <div
          className={cn("h-full rounded-md transition-all", color)}
          style={{ width: `${clamped}%` }}
        />
      </div>
    </div>
  );
}

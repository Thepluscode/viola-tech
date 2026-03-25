import { cn } from "@/lib/utils";
import { formatDate } from "@/lib/utils";
import { ScoreRing } from "./score-ring";
import { FRAMEWORK_LABELS } from "@/types/compliance";
import type { ComplianceScore } from "@/types/compliance";

interface FrameworkCardProps {
  score: ComplianceScore;
  className?: string;
}

export function FrameworkCard({ score, className }: FrameworkCardProps) {
  return (
    <div
      className={cn(
        "rounded-md border border-viola-border bg-viola-surface p-4",
        "hover:border-viola-accent/40 transition-colors",
        className
      )}
    >
      <div className="flex items-start gap-4">
        <ScoreRing
          score={score.overall_score}
          size={90}
          strokeWidth={6}
          label={FRAMEWORK_LABELS[score.framework]}
        />
        <div className="flex-1 min-w-0">
          <h3 className="text-sm font-semibold text-viola-text">
            {FRAMEWORK_LABELS[score.framework]}
          </h3>
          <p className="text-[10px] text-viola-muted mt-0.5">
            Updated {formatDate(score.last_updated)}
          </p>

          <div className="mt-3 grid grid-cols-2 gap-x-4 gap-y-1.5">
            <StatRow label="Passing" value={score.passing} color="text-emerald-400" />
            <StatRow label="Failing" value={score.failing} color="text-red-400" />
            <StatRow label="Partial" value={score.partial} color="text-yellow-400" />
            <StatRow label="Not Assessed" value={score.not_assessed} color="text-viola-muted" />
          </div>

          <div className="mt-3 h-1.5 rounded-full bg-viola-border overflow-hidden flex">
            <div
              className="bg-emerald-500 transition-all"
              style={{ width: `${(score.passing / score.total) * 100}%` }}
            />
            <div
              className="bg-yellow-500 transition-all"
              style={{ width: `${(score.partial / score.total) * 100}%` }}
            />
            <div
              className="bg-red-500 transition-all"
              style={{ width: `${(score.failing / score.total) * 100}%` }}
            />
          </div>
        </div>
      </div>
    </div>
  );
}

function StatRow({
  label,
  value,
  color,
}: {
  label: string;
  value: number;
  color: string;
}) {
  return (
    <div className="flex items-center justify-between text-xs">
      <span className="text-viola-muted">{label}</span>
      <span className={cn("font-mono font-semibold", color)}>{value}</span>
    </div>
  );
}

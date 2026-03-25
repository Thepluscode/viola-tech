import { cn } from "@/lib/utils";
import type { Severity } from "@/types/api";

interface SeverityBadgeProps {
  severity: Severity;
  className?: string;
}

const severityConfig: Record<Severity, { label: string; classes: string }> = {
  critical: {
    label: "Critical",
    classes:
      "bg-severity-critical-bg text-severity-critical border border-severity-critical-border",
  },
  high: {
    label: "High",
    classes:
      "bg-severity-high-bg text-severity-high border border-severity-high-border",
  },
  medium: {
    label: "Medium",
    classes:
      "bg-severity-medium-bg text-severity-medium border border-severity-medium-border",
  },
  low: {
    label: "Low",
    classes:
      "bg-severity-low-bg text-severity-low border border-severity-low-border",
  },
};

export function SeverityBadge({ severity, className }: SeverityBadgeProps) {
  const config = severityConfig[severity];
  return (
    <span
      className={cn(
        "inline-flex items-center px-2 py-0.5 text-xs font-semibold uppercase tracking-wider rounded-md",
        config.classes,
        className
      )}
    >
      {config.label}
    </span>
  );
}

import { cn } from "@/lib/utils";
import type { LucideIcon } from "lucide-react";
import { RiskSparkline } from "@/components/shared/risk-sparkline";

interface KpiCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon: LucideIcon;
  accentColor?: string;
  sparklineValues?: number[];  // Optional 0-100 risk score trend
  sparklineColor?: string;
  className?: string;
}

export function KpiCard({
  title,
  value,
  subtitle,
  icon: Icon,
  accentColor = "text-viola-accent",
  sparklineValues,
  sparklineColor = "#00d4ff",
  className,
}: KpiCardProps) {
  return (
    <div
      className={cn(
        "rounded-md border border-viola-border bg-viola-surface p-4",
        "hover:border-viola-accent/40 transition-colors",
        className
      )}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1 min-w-0">
          <p className="text-xs text-viola-muted uppercase tracking-wider">{title}</p>
          <p className={cn("mt-1.5 text-2xl font-bold font-mono tracking-tight", accentColor)}>
            {value}
          </p>
          {subtitle && (
            <p className="mt-1 text-xs text-viola-muted">{subtitle}</p>
          )}
        </div>
        <div className="flex flex-col items-end gap-1 ml-3">
          <div className={cn("p-2 rounded-md bg-viola-border/50", accentColor)}>
            <Icon className="h-4 w-4" />
          </div>
          {sparklineValues && sparklineValues.length >= 2 && (
            <RiskSparkline
              values={sparklineValues}
              color={sparklineColor}
              width={64}
              height={20}
              className="opacity-80"
            />
          )}
        </div>
      </div>
    </div>
  );
}

import { cn } from "@/lib/utils";

interface ConfidenceBarProps {
  value: number;
  showLabel?: boolean;
  className?: string;
}

export function ConfidenceBar({ value, showLabel = true, className }: ConfidenceBarProps) {
  const clamped = Math.min(100, Math.max(0, value));
  const color =
    clamped >= 80
      ? "bg-viola-accent"
      : clamped >= 50
        ? "bg-severity-medium"
        : "bg-viola-muted";

  return (
    <div className={cn("flex items-center gap-2", className)}>
      {showLabel && (
        <span className="text-xs font-mono text-viola-muted w-6 text-right">{clamped}</span>
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

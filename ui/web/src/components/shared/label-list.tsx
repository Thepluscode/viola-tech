import { cn } from "@/lib/utils";
import { Tag } from "lucide-react";
import type { Labels } from "@/types/api";

interface LabelListProps {
  labels: Labels;
  className?: string;
}

export function LabelList({ labels, className }: LabelListProps) {
  const entries = Object.entries(labels);
  if (entries.length === 0) {
    return <span className="text-xs text-viola-muted italic">No labels</span>;
  }
  return (
    <div className={cn("flex flex-wrap gap-1.5", className)}>
      {entries.map(([key, value]) => (
        <span
          key={key}
          className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-viola-surface border border-viola-border text-xs text-viola-muted"
        >
          <Tag className="h-3 w-3 shrink-0" />
          <span className="text-viola-muted">{key}:</span>
          <span className="text-viola-text">{value}</span>
        </span>
      ))}
    </div>
  );
}

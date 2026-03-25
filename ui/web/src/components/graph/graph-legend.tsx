import { cn } from "@/lib/utils";
import { NODE_TYPE_LABELS, EDGE_TYPE_LABELS } from "@/types/graph";
import type { NodeType, EdgeType } from "@/types/graph";

const NODE_COLORS: Record<NodeType, string> = {
  device: "#3b82f6",
  user: "#8b5cf6",
  service: "#10b981",
  "cloud-resource": "#f59e0b",
  "crown-jewel": "#ef4444",
};

const EDGE_COLORS: Record<EdgeType, string> = {
  auth: "#8b5cf6",
  network: "#3b82f6",
  process: "#10b981",
  "cloud-api": "#f59e0b",
  "lateral-movement": "#ef4444",
};

export function GraphLegend({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        "rounded-md border border-viola-border bg-viola-surface p-3",
        className
      )}
    >
      <p className="text-[10px] text-viola-muted uppercase tracking-wider mb-2">
        Node Types
      </p>
      <div className="flex flex-wrap gap-3 mb-3">
        {(Object.keys(NODE_TYPE_LABELS) as NodeType[]).map((type) => (
          <div key={type} className="flex items-center gap-1.5">
            <div
              className="w-3 h-3 rounded-full"
              style={{ backgroundColor: NODE_COLORS[type] }}
            />
            <span className="text-[10px] text-viola-muted">
              {NODE_TYPE_LABELS[type]}
            </span>
          </div>
        ))}
      </div>

      <p className="text-[10px] text-viola-muted uppercase tracking-wider mb-2">
        Edge Types
      </p>
      <div className="flex flex-wrap gap-3">
        {(Object.keys(EDGE_TYPE_LABELS) as EdgeType[]).map((type) => (
          <div key={type} className="flex items-center gap-1.5">
            <div
              className="w-4 h-0.5 rounded"
              style={{ backgroundColor: EDGE_COLORS[type] }}
            />
            <span className="text-[10px] text-viola-muted">
              {EDGE_TYPE_LABELS[type]}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

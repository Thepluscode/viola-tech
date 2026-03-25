"use client";

import { cn } from "@/lib/utils";
import type { GraphNode, GraphEdge } from "@/types/graph";
import { NODE_TYPE_LABELS, EDGE_TYPE_LABELS } from "@/types/graph";
import { formatRelative } from "@/lib/utils";
import { X } from "lucide-react";

interface NodeDetailPanelProps {
  node: GraphNode;
  edges: GraphEdge[];
  allNodes: GraphNode[];
  onClose: () => void;
}

export function NodeDetailPanel({
  node,
  edges,
  allNodes,
  onClose,
}: NodeDetailPanelProps) {
  const connectedEdges = edges.filter(
    (e) => e.source === node.id || e.target === node.id
  );
  const nodeMap = new Map(allNodes.map((n) => [n.id, n]));

  const riskColor =
    node.risk_score >= 80
      ? "text-red-400"
      : node.risk_score >= 50
        ? "text-yellow-400"
        : "text-emerald-400";

  return (
    <div className="rounded-md border border-viola-border bg-viola-surface p-4 w-80">
      <div className="flex items-start justify-between mb-3">
        <div>
          <h3 className="text-sm font-semibold text-viola-text truncate max-w-[240px]">
            {node.label}
          </h3>
          <span className="text-[10px] text-viola-muted">
            {NODE_TYPE_LABELS[node.type]}
            {node.is_crown_jewel && " — Crown Jewel"}
          </span>
        </div>
        <button
          onClick={onClose}
          className="p-1 hover:bg-viola-border/50 rounded transition-colors"
        >
          <X className="h-3.5 w-3.5 text-viola-muted" />
        </button>
      </div>

      {/* Risk score */}
      <div className="flex items-center gap-3 mb-3">
        <span className="text-xs text-viola-muted">Risk Score</span>
        <span className={cn("text-lg font-bold font-mono", riskColor)}>
          {node.risk_score}
        </span>
        <div className="flex-1 h-1.5 rounded-full bg-viola-border overflow-hidden">
          <div
            className={cn(
              "h-full rounded-full transition-all",
              node.risk_score >= 80
                ? "bg-red-500"
                : node.risk_score >= 50
                  ? "bg-yellow-500"
                  : "bg-emerald-500"
            )}
            style={{ width: `${node.risk_score}%` }}
          />
        </div>
      </div>

      {/* Metadata */}
      {Object.keys(node.metadata).length > 0 && (
        <div className="mb-3">
          <p className="text-[10px] text-viola-muted uppercase tracking-wider mb-1">
            Metadata
          </p>
          <div className="space-y-0.5">
            {Object.entries(node.metadata).map(([k, v]) => (
              <div key={k} className="flex items-center justify-between text-xs">
                <span className="text-viola-muted">{k}</span>
                <span className="text-viola-text font-mono">{v}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Connections */}
      <div>
        <p className="text-[10px] text-viola-muted uppercase tracking-wider mb-1">
          Connections ({connectedEdges.length})
        </p>
        <div className="space-y-1 max-h-48 overflow-y-auto">
          {connectedEdges.map((edge) => {
            const otherId =
              edge.source === node.id ? edge.target : edge.source;
            const otherNode = nodeMap.get(otherId);
            const direction = edge.source === node.id ? "→" : "←";

            return (
              <div
                key={edge.id}
                className="flex items-center gap-2 text-xs px-2 py-1 rounded bg-viola-bg/50"
              >
                <span className="text-viola-muted font-mono">{direction}</span>
                <span className="text-viola-text truncate flex-1">
                  {otherNode?.label ?? otherId}
                </span>
                <span className="text-[10px] text-viola-muted">
                  {EDGE_TYPE_LABELS[edge.type]}
                </span>
                <span className="text-[10px] text-viola-muted font-mono">
                  {edge.event_count}
                </span>
              </div>
            );
          })}
        </div>
      </div>

      {/* Entity IDs */}
      {node.entity_ids.length > 0 && (
        <div className="mt-3">
          <p className="text-[10px] text-viola-muted uppercase tracking-wider mb-1">
            Entity IDs
          </p>
          {node.entity_ids.map((eid) => (
            <p key={eid} className="text-[10px] text-viola-accent font-mono truncate">
              {eid}
            </p>
          ))}
        </div>
      )}
    </div>
  );
}

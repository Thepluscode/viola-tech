"use client";

import { useState } from "react";
import { ForceGraph } from "@/components/graph/force-graph";
import { NodeDetailPanel } from "@/components/graph/node-detail-panel";
import { GraphLegend } from "@/components/graph/graph-legend";
import { MOCK_ATTACK_GRAPH } from "@/lib/mock-graph";
import type { GraphNode } from "@/types/graph";
import { formatDate } from "@/lib/utils";
import { GitBranch } from "lucide-react";

export default function GraphPage() {
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);
  const graph = MOCK_ATTACK_GRAPH;

  const crownJewels = graph.nodes.filter((n) => n.is_crown_jewel);
  const highRiskCount = graph.nodes.filter((n) => n.risk_score >= 70).length;

  return (
    <div className="px-6 py-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-bold text-viola-text">Attack Graph</h1>
          <p className="text-xs text-viola-muted mt-0.5">
            {graph.nodes.length} nodes, {graph.edges.length} edges — Updated{" "}
            {formatDate(graph.updated_at)}
          </p>
        </div>
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2 text-xs bg-red-950/40 border border-red-900 px-3 py-1.5 rounded-md text-red-400">
            <GitBranch className="h-3.5 w-3.5" />
            {highRiskCount} high-risk nodes
          </div>
          <div className="flex items-center gap-2 text-xs bg-viola-surface border border-viola-border px-3 py-1.5 rounded-md text-viola-muted">
            {crownJewels.length} crown jewels
          </div>
        </div>
      </div>

      <GraphLegend className="mb-4" />

      <div className="flex gap-4">
        <div className="flex-1">
          <ForceGraph
            nodes={graph.nodes}
            edges={graph.edges}
            width={selectedNode ? 700 : 900}
            height={600}
            onNodeClick={setSelectedNode}
          />
        </div>

        {selectedNode && (
          <NodeDetailPanel
            node={selectedNode}
            edges={graph.edges}
            allNodes={graph.nodes}
            onClose={() => setSelectedNode(null)}
          />
        )}
      </div>

      {/* Crown jewels summary */}
      {crownJewels.length > 0 && (
        <div className="mt-6 rounded-md border border-viola-border bg-viola-surface p-4">
          <h3 className="text-sm font-semibold text-viola-text mb-3">Crown Jewel Assets</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
            {crownJewels.map((node) => (
              <button
                key={node.id}
                onClick={() => setSelectedNode(node)}
                className="text-left p-3 rounded-md bg-viola-bg border border-viola-border hover:border-red-900/60 transition-colors"
              >
                <p className="text-xs text-viola-text font-medium truncate">
                  {node.label}
                </p>
                <div className="flex items-center justify-between mt-1">
                  <span className="text-[10px] text-viola-muted">
                    Risk: {node.risk_score}
                  </span>
                  <span className="text-[10px] text-red-400 font-mono">
                    {node.entity_ids.length} entities
                  </span>
                </div>
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { cn } from "@/lib/utils";
import type { GraphNode, GraphEdge, NodeType, EdgeType } from "@/types/graph";

interface ForceGraphProps {
  nodes: GraphNode[];
  edges: GraphEdge[];
  width?: number;
  height?: number;
  onNodeClick?: (node: GraphNode) => void;
  className?: string;
}

// Simple force simulation in pure JS — no D3 dependency
interface SimNode extends GraphNode {
  x: number;
  y: number;
  vx: number;
  vy: number;
  fx?: number;
  fy?: number;
}

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

const NODE_RADIUS: Record<NodeType, number> = {
  device: 8,
  user: 7,
  service: 9,
  "cloud-resource": 8,
  "crown-jewel": 11,
};

export function ForceGraph({
  nodes,
  edges,
  width = 800,
  height = 600,
  onNodeClick,
  className,
}: ForceGraphProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [simNodes, setSimNodes] = useState<SimNode[]>([]);
  const [hoveredNode, setHoveredNode] = useState<string | null>(null);
  const [dragNode, setDragNode] = useState<string | null>(null);
  const animRef = useRef<number>();
  const simNodesRef = useRef<SimNode[]>([]);

  // Initialize node positions
  useEffect(() => {
    const initial: SimNode[] = nodes.map((n, i) => ({
      ...n,
      x: width / 2 + (Math.cos((i / nodes.length) * Math.PI * 2) * width) / 3,
      y: height / 2 + (Math.sin((i / nodes.length) * Math.PI * 2) * height) / 3,
      vx: 0,
      vy: 0,
    }));
    simNodesRef.current = initial;
    setSimNodes([...initial]);
  }, [nodes, width, height]);

  // Force simulation tick
  useEffect(() => {
    let iteration = 0;
    const maxIter = 300;
    const alpha = 0.3;
    const decay = 0.995;

    function tick() {
      if (iteration >= maxIter) return;
      iteration++;

      const sn = simNodesRef.current;
      const currentAlpha = alpha * Math.pow(decay, iteration);
      if (currentAlpha < 0.001) return;

      const nodeMap = new Map(sn.map((n) => [n.id, n]));

      // Repulsion between all nodes
      for (let i = 0; i < sn.length; i++) {
        for (let j = i + 1; j < sn.length; j++) {
          const a = sn[i];
          const b = sn[j];
          let dx = b.x - a.x;
          let dy = b.y - a.y;
          let dist = Math.sqrt(dx * dx + dy * dy) || 1;
          const force = (150 * currentAlpha) / (dist * dist);
          const fx = (dx / dist) * force;
          const fy = (dy / dist) * force;
          if (!a.fx) a.vx -= fx;
          if (!a.fy) a.vy -= fy;
          if (!b.fx) b.vx += fx;
          if (!b.fy) b.vy += fy;
        }
      }

      // Attraction along edges
      for (const edge of edges) {
        const source = nodeMap.get(edge.source);
        const target = nodeMap.get(edge.target);
        if (!source || !target) continue;
        const dx = target.x - source.x;
        const dy = target.y - source.y;
        const dist = Math.sqrt(dx * dx + dy * dy) || 1;
        const force = (dist - 120) * 0.01 * currentAlpha;
        const fx = (dx / dist) * force;
        const fy = (dy / dist) * force;
        if (!source.fx) source.vx += fx;
        if (!source.fy) source.vy += fy;
        if (!target.fx) target.vx -= fx;
        if (!target.fy) target.vy -= fy;
      }

      // Center gravity
      for (const n of sn) {
        if (!n.fx) n.vx += (width / 2 - n.x) * 0.001 * currentAlpha;
        if (!n.fy) n.vy += (height / 2 - n.y) * 0.001 * currentAlpha;
      }

      // Apply velocity with damping
      for (const n of sn) {
        if (n.fx !== undefined) {
          n.x = n.fx;
          n.vx = 0;
        } else {
          n.vx *= 0.6;
          n.x += n.vx;
        }
        if (n.fy !== undefined) {
          n.y = n.fy;
          n.vy = 0;
        } else {
          n.vy *= 0.6;
          n.y += n.vy;
        }
        // Clamp to bounds
        n.x = Math.max(20, Math.min(width - 20, n.x));
        n.y = Math.max(20, Math.min(height - 20, n.y));
      }

      setSimNodes([...sn]);
      animRef.current = requestAnimationFrame(tick);
    }

    animRef.current = requestAnimationFrame(tick);
    return () => {
      if (animRef.current) cancelAnimationFrame(animRef.current);
    };
  }, [edges, width, height]);

  const nodeMap = new Map(simNodes.map((n) => [n.id, n]));

  const handleMouseDown = useCallback(
    (nodeId: string, e: React.MouseEvent) => {
      e.preventDefault();
      setDragNode(nodeId);
      const node = simNodesRef.current.find((n) => n.id === nodeId);
      if (node) {
        node.fx = node.x;
        node.fy = node.y;
      }
    },
    []
  );

  const handleMouseMove = useCallback(
    (e: React.MouseEvent) => {
      if (!dragNode || !svgRef.current) return;
      const rect = svgRef.current.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;
      const node = simNodesRef.current.find((n) => n.id === dragNode);
      if (node) {
        node.fx = x;
        node.fy = y;
        node.x = x;
        node.y = y;
        setSimNodes([...simNodesRef.current]);
      }
    },
    [dragNode]
  );

  const handleMouseUp = useCallback(() => {
    if (dragNode) {
      const node = simNodesRef.current.find((n) => n.id === dragNode);
      if (node) {
        delete node.fx;
        delete node.fy;
      }
    }
    setDragNode(null);
  }, [dragNode]);

  return (
    <svg
      ref={svgRef}
      width={width}
      height={height}
      className={cn("bg-viola-bg rounded-md border border-viola-border", className)}
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onMouseLeave={handleMouseUp}
    >
      <defs>
        <marker
          id="arrowhead"
          markerWidth="8"
          markerHeight="6"
          refX="8"
          refY="3"
          orient="auto"
        >
          <polygon points="0 0, 8 3, 0 6" fill="#475569" />
        </marker>
      </defs>

      {/* Edges */}
      {edges.map((edge) => {
        const source = nodeMap.get(edge.source);
        const target = nodeMap.get(edge.target);
        if (!source || !target) return null;
        const isHighlighted =
          hoveredNode === edge.source || hoveredNode === edge.target;

        return (
          <line
            key={edge.id}
            x1={source.x}
            y1={source.y}
            x2={target.x}
            y2={target.y}
            stroke={EDGE_COLORS[edge.type]}
            strokeWidth={isHighlighted ? 2 : 1}
            strokeOpacity={isHighlighted ? 0.9 : 0.3}
            markerEnd="url(#arrowhead)"
          />
        );
      })}

      {/* Nodes */}
      {simNodes.map((node) => {
        const r = NODE_RADIUS[node.type] ?? 8;
        const color = node.is_crown_jewel
          ? NODE_COLORS["crown-jewel"]
          : NODE_COLORS[node.type];
        const isHovered = hoveredNode === node.id;

        return (
          <g
            key={node.id}
            className="cursor-pointer"
            onMouseEnter={() => setHoveredNode(node.id)}
            onMouseLeave={() => setHoveredNode(null)}
            onMouseDown={(e) => handleMouseDown(node.id, e)}
            onClick={() => onNodeClick?.(node)}
          >
            {/* Risk glow for high-risk nodes */}
            {node.risk_score >= 70 && (
              <circle
                cx={node.x}
                cy={node.y}
                r={r + 6}
                fill="none"
                stroke={color}
                strokeWidth={1}
                strokeOpacity={0.3}
                className="animate-pulse"
              />
            )}
            {/* Crown jewel ring */}
            {node.is_crown_jewel && (
              <circle
                cx={node.x}
                cy={node.y}
                r={r + 3}
                fill="none"
                stroke="#ef4444"
                strokeWidth={1.5}
                strokeDasharray="3 2"
              />
            )}
            <circle
              cx={node.x}
              cy={node.y}
              r={isHovered ? r + 2 : r}
              fill={color}
              fillOpacity={0.8}
              stroke={isHovered ? "#fff" : color}
              strokeWidth={isHovered ? 2 : 1}
            />
            {/* Label */}
            {isHovered && (
              <text
                x={node.x}
                y={node.y - r - 6}
                textAnchor="middle"
                className="text-[10px] fill-viola-text"
                style={{ pointerEvents: "none" }}
              >
                {node.label}
              </text>
            )}
          </g>
        );
      })}
    </svg>
  );
}

export type NodeType = "device" | "user" | "service" | "cloud-resource" | "crown-jewel";

export type EdgeType = "auth" | "network" | "process" | "cloud-api" | "lateral-movement";

export interface GraphNode {
  id: string;
  label: string;
  type: NodeType;
  risk_score: number; // 0-100
  entity_ids: string[];
  is_crown_jewel: boolean;
  metadata: Record<string, string>;
}

export interface GraphEdge {
  id: string;
  source: string;
  target: string;
  type: EdgeType;
  weight: number; // 0-1 strength
  last_seen: string;
  event_count: number;
  labels: Record<string, string>;
}

export interface AttackGraph {
  nodes: GraphNode[];
  edges: GraphEdge[];
  updated_at: string;
}

export const NODE_TYPE_LABELS: Record<NodeType, string> = {
  device: "Device",
  user: "User",
  service: "Service",
  "cloud-resource": "Cloud Resource",
  "crown-jewel": "Crown Jewel",
};

export const EDGE_TYPE_LABELS: Record<EdgeType, string> = {
  auth: "Authentication",
  network: "Network",
  process: "Process",
  "cloud-api": "Cloud API",
  "lateral-movement": "Lateral Movement",
};

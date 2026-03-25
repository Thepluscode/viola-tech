// Status values match backend exactly — backend uses "ack" not "acknowledged"
export type Severity = "critical" | "high" | "medium" | "low";
export type IncidentStatus = "open" | "ack" | "closed";
export type AlertStatus = "open" | "ack" | "closed";

// Backend returns labels as map[string]string e.g. {"environment": "production"}
export type Labels = Record<string, string>;

export interface Incident {
  tenant_id: string;
  incident_id: string;
  correlated_group_id: string;
  created_at: string;
  updated_at: string;
  status: IncidentStatus;
  severity: Severity;
  max_risk_score: number;      // 0-100 float
  max_confidence: number;      // 0-1 float
  mitre_tactic: string | null;
  mitre_technique: string | null;   // bare technique string, not an object
  labels: Labels;
  assigned_to: string | null;
  closure_reason: string | null;
  request_id: string | null;
  alert_count: number;
  hit_count: number;
  entity_ids: string[];
  alert_ids: string[];
  detection_hit_ids: string[];
}

export interface Alert {
  tenant_id: string;
  alert_id: string;
  created_at: string;
  updated_at: string;
  status: AlertStatus;
  severity: Severity;
  confidence: number;      // 0-1 float
  risk_score: number;      // 0-100 float
  title: string;
  description: string;
  mitre_tactic: string | null;
  mitre_technique: string | null;   // bare technique string
  labels: Labels;
  assigned_to: string | null;
  closure_reason: string | null;
  request_id: string | null;
  entity_ids: string[];
  detection_hit_ids: string[];
}

// Backend response envelopes — keys are "incidents"/"alerts", not "data"
export interface IncidentListResponse {
  incidents: Incident[];
  count: number;
}

export interface AlertListResponse {
  alerts: Alert[];
  count: number;
}

// Filter params — backend uses limit/offset, not page/page_size
export interface IncidentFilters {
  status?: IncidentStatus;
  severity?: Severity;
  limit?: number;
  offset?: number;
}

export interface AlertFilters {
  status?: AlertStatus;
  severity?: Severity;
  incident_id?: string;     // UI-only filter, not a backend param
  limit?: number;
  offset?: number;
}

// PATCH payloads — status must be "ack" not "acknowledged"
export interface UpdateIncidentPayload {
  status?: IncidentStatus;
  assigned_to?: string | null;
  closure_reason?: string | null;
}

export interface UpdateAlertPayload {
  status?: AlertStatus;
  assigned_to?: string | null;
  closure_reason?: string | null;
}

// UI display helpers — convert status to display label
export const STATUS_LABELS: Record<IncidentStatus | AlertStatus, string> = {
  open: "Open",
  ack: "Acknowledged",
  closed: "Closed",
};

export const PAGE_SIZE = 20;

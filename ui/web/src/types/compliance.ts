export type ComplianceFramework = "soc2" | "hipaa" | "nist-csf" | "pci-dss";

export type ControlStatus = "passing" | "failing" | "partial" | "not-assessed";

export interface ComplianceControl {
  control_id: string;
  framework: ComplianceFramework;
  category: string;
  title: string;
  description: string;
  status: ControlStatus;
  evidence_count: number;
  last_assessed: string;
  owner: string | null;
  notes: string | null;
}

export interface ComplianceScore {
  framework: ComplianceFramework;
  overall_score: number; // 0-100
  passing: number;
  failing: number;
  partial: number;
  not_assessed: number;
  total: number;
  last_updated: string;
}

export interface AuditEvent {
  event_id: string;
  timestamp: string;
  actor: string;
  action: string;
  resource: string;
  framework: ComplianceFramework;
  control_id: string | null;
  result: "pass" | "fail" | "info";
}

export const FRAMEWORK_LABELS: Record<ComplianceFramework, string> = {
  "soc2": "SOC 2 Type II",
  "hipaa": "HIPAA",
  "nist-csf": "NIST CSF",
  "pci-dss": "PCI DSS",
};

export const CONTROL_STATUS_LABELS: Record<ControlStatus, string> = {
  passing: "Passing",
  failing: "Failing",
  partial: "Partial",
  "not-assessed": "Not Assessed",
};

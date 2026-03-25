import Link from "next/link";
import { SeverityBadge } from "@/components/shared/severity-badge";
import { StatusBadge } from "@/components/shared/status-badge";
import { RiskScoreBar } from "@/components/shared/risk-score-bar";
import { MitreTag } from "@/components/shared/mitre-tag";
import { formatRelative } from "@/lib/utils";
import type { Incident } from "@/types/api";

interface IncidentsTableProps {
  incidents: Incident[];
}

export function IncidentsTable({ incidents }: IncidentsTableProps) {
  if (incidents.length === 0) {
    return (
      <div className="rounded-md border border-viola-border bg-viola-surface px-4 py-12 text-center">
        <p className="text-sm text-viola-muted">No incidents match the current filters.</p>
      </div>
    );
  }

  return (
    <div className="rounded-md border border-viola-border overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-viola-surface border-b border-viola-border">
              <th className="px-4 py-3 text-left text-xs font-semibold text-viola-muted uppercase tracking-wider">ID</th>
              <th className="px-4 py-3 text-left text-xs font-semibold text-viola-muted uppercase tracking-wider">Severity</th>
              <th className="px-4 py-3 text-left text-xs font-semibold text-viola-muted uppercase tracking-wider">Status</th>
              <th className="px-4 py-3 text-left text-xs font-semibold text-viola-muted uppercase tracking-wider w-36">Risk</th>
              <th className="px-4 py-3 text-left text-xs font-semibold text-viola-muted uppercase tracking-wider">MITRE Tactic</th>
              <th className="px-4 py-3 text-left text-xs font-semibold text-viola-muted uppercase tracking-wider">Alerts</th>
              <th className="px-4 py-3 text-left text-xs font-semibold text-viola-muted uppercase tracking-wider">Assigned</th>
              <th className="px-4 py-3 text-left text-xs font-semibold text-viola-muted uppercase tracking-wider">Updated</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-viola-border">
            {incidents.map((incident) => (
              <tr key={incident.incident_id} className="bg-viola-bg hover:bg-viola-surface/60 transition-colors">
                <td className="px-4 py-3">
                  <Link
                    href={`/incidents/${incident.incident_id}`}
                    className="font-mono text-xs text-viola-accent hover:underline"
                  >
                    {incident.incident_id}
                  </Link>
                </td>
                <td className="px-4 py-3">
                  <SeverityBadge severity={incident.severity} />
                </td>
                <td className="px-4 py-3">
                  <StatusBadge status={incident.status} />
                </td>
                <td className="px-4 py-3 w-36">
                  <RiskScoreBar score={incident.max_risk_score} />
                </td>
                <td className="px-4 py-3">
                  <MitreTag
                    id={incident.mitre_technique ?? undefined}
                    tactic={incident.mitre_tactic ?? undefined}
                  />
                </td>
                <td className="px-4 py-3">
                  <span className="font-mono text-xs text-viola-muted">{incident.alert_count}</span>
                </td>
                <td className="px-4 py-3">
                  <span className="text-xs text-viola-muted truncate max-w-[120px] block">
                    {incident.assigned_to ?? <span className="italic text-viola-border">Unassigned</span>}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <span className="text-xs text-viola-muted">{formatRelative(incident.updated_at)}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

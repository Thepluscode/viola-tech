import Link from "next/link";
import { SeverityBadge } from "@/components/shared/severity-badge";
import { StatusBadge } from "@/components/shared/status-badge";
import { RiskScoreBar } from "@/components/shared/risk-score-bar";
import { ConfidenceBar } from "@/components/shared/confidence-bar";
import { MitreTag } from "@/components/shared/mitre-tag";
import { formatRelative, truncate } from "@/lib/utils";
import type { Alert } from "@/types/api";

interface AlertsTableProps {
  alerts: Alert[];
}

export function AlertsTable({ alerts }: AlertsTableProps) {
  if (alerts.length === 0) {
    return (
      <div className="rounded-md border border-viola-border bg-viola-surface px-4 py-12 text-center">
        <p className="text-sm text-viola-muted">No alerts match the current filters.</p>
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
              <th className="px-4 py-3 text-left text-xs font-semibold text-viola-muted uppercase tracking-wider">Title</th>
              <th className="px-4 py-3 text-left text-xs font-semibold text-viola-muted uppercase tracking-wider">Severity</th>
              <th className="px-4 py-3 text-left text-xs font-semibold text-viola-muted uppercase tracking-wider">Status</th>
              <th className="px-4 py-3 text-left text-xs font-semibold text-viola-muted uppercase tracking-wider w-36">Risk</th>
              <th className="px-4 py-3 text-left text-xs font-semibold text-viola-muted uppercase tracking-wider w-36">Confidence</th>
              <th className="px-4 py-3 text-left text-xs font-semibold text-viola-muted uppercase tracking-wider">MITRE</th>
              <th className="px-4 py-3 text-left text-xs font-semibold text-viola-muted uppercase tracking-wider">Updated</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-viola-border">
            {alerts.map((alert) => (
              <tr key={alert.alert_id} className="bg-viola-bg hover:bg-viola-surface/60 transition-colors">
                <td className="px-4 py-3">
                  <Link
                    href={`/alerts/${alert.alert_id}`}
                    className="font-mono text-xs text-viola-accent hover:underline"
                  >
                    {alert.alert_id}
                  </Link>
                </td>
                <td className="px-4 py-3 max-w-[240px]">
                  <Link href={`/alerts/${alert.alert_id}`} className="text-xs text-viola-text hover:text-viola-accent transition-colors">
                    {truncate(alert.title, 50)}
                  </Link>
                </td>
                <td className="px-4 py-3">
                  <SeverityBadge severity={alert.severity} />
                </td>
                <td className="px-4 py-3">
                  <StatusBadge status={alert.status} />
                </td>
                <td className="px-4 py-3 w-36">
                  <RiskScoreBar score={alert.risk_score} />
                </td>
                <td className="px-4 py-3 w-36">
                  {/* confidence is 0-1, convert to 0-100 for the bar */}
                  <ConfidenceBar value={alert.confidence * 100} />
                </td>
                <td className="px-4 py-3">
                  <MitreTag
                    id={alert.mitre_technique ?? undefined}
                    tactic={alert.mitre_tactic ?? undefined}
                  />
                </td>
                <td className="px-4 py-3">
                  <span className="text-xs text-viola-muted">{formatRelative(alert.updated_at)}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

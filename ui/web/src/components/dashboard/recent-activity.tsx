import Link from "next/link";
import { SeverityBadge } from "@/components/shared/severity-badge";
import { StatusBadge } from "@/components/shared/status-badge";
import { formatRelative } from "@/lib/utils";
import type { Incident, Alert } from "@/types/api";

interface RecentActivityProps {
  incidents: Incident[];
  alerts: Alert[];
}

export function RecentActivity({ incidents, alerts }: RecentActivityProps) {
  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-6">
      {/* Recent Incidents */}
      <div className="rounded-md border border-viola-border bg-viola-surface">
        <div className="px-4 py-3 border-b border-viola-border flex items-center justify-between">
          <h2 className="text-sm font-semibold text-viola-text">Recent Incidents</h2>
          <Link href="/incidents" className="text-xs text-viola-accent hover:underline">
            View all →
          </Link>
        </div>
        <div className="divide-y divide-viola-border">
          {incidents.slice(0, 5).map((incident) => (
            <Link
              key={incident.incident_id}
              href={`/incidents/${incident.incident_id}`}
              className="flex items-center gap-3 px-4 py-3 hover:bg-viola-border/20 transition-colors"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="font-mono text-xs text-viola-accent">{incident.incident_id}</span>
                  <SeverityBadge severity={incident.severity} />
                  <StatusBadge status={incident.status} />
                </div>
                <p className="text-xs text-viola-muted mt-0.5 truncate">
                  {incident.mitre_tactic ?? "Unknown tactic"} · Risk {Math.round(incident.max_risk_score)}
                </p>
              </div>
              <span className="text-xs text-viola-muted shrink-0">
                {formatRelative(incident.updated_at)}
              </span>
            </Link>
          ))}
          {incidents.length === 0 && (
            <p className="px-4 py-6 text-xs text-viola-muted text-center">No recent incidents</p>
          )}
        </div>
      </div>

      {/* Recent Alerts */}
      <div className="rounded-md border border-viola-border bg-viola-surface">
        <div className="px-4 py-3 border-b border-viola-border flex items-center justify-between">
          <h2 className="text-sm font-semibold text-viola-text">Recent Alerts</h2>
          <Link href="/alerts" className="text-xs text-viola-accent hover:underline">
            View all →
          </Link>
        </div>
        <div className="divide-y divide-viola-border">
          {alerts.slice(0, 5).map((alert) => (
            <Link
              key={alert.alert_id}
              href={`/alerts/${alert.alert_id}`}
              className="flex items-center gap-3 px-4 py-3 hover:bg-viola-border/20 transition-colors"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="font-mono text-xs text-viola-muted">{alert.alert_id}</span>
                  <SeverityBadge severity={alert.severity} />
                </div>
                <p className="text-xs text-viola-text mt-0.5 truncate">{alert.title}</p>
              </div>
              <span className="text-xs text-viola-muted shrink-0">
                {formatRelative(alert.updated_at)}
              </span>
            </Link>
          ))}
          {alerts.length === 0 && (
            <p className="px-4 py-6 text-xs text-viola-muted text-center">No recent alerts</p>
          )}
        </div>
      </div>
    </div>
  );
}

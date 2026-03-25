import Link from "next/link";
import { SeverityBadge } from "@/components/shared/severity-badge";
import { StatusBadge } from "@/components/shared/status-badge";
import { RiskScoreBar } from "@/components/shared/risk-score-bar";
import { MitreTag } from "@/components/shared/mitre-tag";
import { LabelList } from "@/components/shared/label-list";
import { EntityList } from "@/components/shared/entity-list";
import { formatDate, formatRelative } from "@/lib/utils";
import type { Incident, Alert } from "@/types/api";
import { ArrowLeft, User, Calendar, AlertCircle } from "lucide-react";

interface IncidentDetailProps {
  incident: Incident;
  relatedAlerts: Alert[];
}

export function IncidentDetail({ incident, relatedAlerts }: IncidentDetailProps) {
  return (
    <div className="max-w-5xl">
      <Link
        href="/incidents"
        className="inline-flex items-center gap-1.5 text-xs text-viola-muted hover:text-viola-accent mb-4 transition-colors"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Back to incidents
      </Link>

      {/* Header */}
      <div className="flex items-start justify-between flex-wrap gap-4 mb-6">
        <div>
          <div className="flex items-center gap-3 flex-wrap">
            <h1 className="font-mono text-lg font-bold text-viola-accent">{incident.incident_id}</h1>
            <SeverityBadge severity={incident.severity} />
            <StatusBadge status={incident.status} />
          </div>
          {incident.mitre_tactic && (
            <p className="text-sm text-viola-muted mt-1">
              {incident.mitre_tactic}
              {incident.mitre_technique && ` · ${incident.mitre_technique}`}
            </p>
          )}
        </div>
        <div className="text-right text-xs text-viola-muted space-y-1">
          <div className="flex items-center gap-1.5 justify-end">
            <Calendar className="h-3.5 w-3.5" />
            Created {formatDate(incident.created_at)}
          </div>
          <div className="flex items-center gap-1.5 justify-end">
            <Calendar className="h-3.5 w-3.5" />
            Updated {formatRelative(incident.updated_at)}
          </div>
          {incident.assigned_to && (
            <div className="flex items-center gap-1.5 justify-end">
              <User className="h-3.5 w-3.5" />
              {incident.assigned_to}
            </div>
          )}
        </div>
      </div>

      {/* Details grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
        {/* Risk & MITRE */}
        <div className="rounded-md border border-viola-border bg-viola-surface p-4 space-y-4">
          <h2 className="text-xs font-semibold text-viola-muted uppercase tracking-wider">
            Threat Intelligence
          </h2>
          <div>
            <p className="text-xs text-viola-muted mb-1.5">Risk Score</p>
            <RiskScoreBar score={incident.max_risk_score} />
          </div>
          <div>
            <p className="text-xs text-viola-muted mb-1.5">Confidence</p>
            <p className="font-mono text-sm text-viola-text">
              {Math.round(incident.max_confidence * 100)}%
            </p>
          </div>
          {(incident.mitre_tactic || incident.mitre_technique) && (
            <div>
              <p className="text-xs text-viola-muted mb-1.5">MITRE ATT&CK</p>
              <div className="flex flex-col gap-1.5">
                <MitreTag
                  id={incident.mitre_technique ?? undefined}
                  tactic={incident.mitre_tactic ?? undefined}
                />
                {incident.mitre_tactic && (
                  <span className="text-xs text-viola-muted">{incident.mitre_tactic}</span>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Affected Entities */}
        <div className="rounded-md border border-viola-border bg-viola-surface p-4 space-y-4">
          <h2 className="text-xs font-semibold text-viola-muted uppercase tracking-wider">
            Affected Entities
          </h2>
          <EntityList entities={incident.entity_ids} />
        </div>

        {/* Labels */}
        <div className="rounded-md border border-viola-border bg-viola-surface p-4 space-y-3">
          <h2 className="text-xs font-semibold text-viola-muted uppercase tracking-wider">Labels</h2>
          <LabelList labels={incident.labels} />
        </div>

        {/* Stats */}
        <div className="rounded-md border border-viola-border bg-viola-surface p-4 space-y-3">
          <h2 className="text-xs font-semibold text-viola-muted uppercase tracking-wider">
            Incident Stats
          </h2>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <p className="text-xs text-viola-muted">Alert Count</p>
              <p className="font-mono text-lg font-bold text-viola-accent">{incident.alert_count}</p>
            </div>
            <div>
              <p className="text-xs text-viola-muted">Hit Count</p>
              <p className="font-mono text-lg font-bold text-viola-text">{incident.hit_count}</p>
            </div>
            <div>
              <p className="text-xs text-viola-muted">Assigned To</p>
              <p className="text-sm text-viola-text mt-0.5">
                {incident.assigned_to ?? <span className="italic text-viola-muted">Unassigned</span>}
              </p>
            </div>
            {incident.closure_reason && (
              <div>
                <p className="text-xs text-viola-muted">Closure Reason</p>
                <p className="text-xs text-viola-text mt-0.5">{incident.closure_reason}</p>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Related Alerts */}
      {relatedAlerts.length > 0 && (
        <div className="rounded-md border border-viola-border bg-viola-surface">
          <div className="px-4 py-3 border-b border-viola-border">
            <h2 className="text-sm font-semibold text-viola-text flex items-center gap-2">
              <AlertCircle className="h-4 w-4 text-viola-accent" />
              Related Alerts ({relatedAlerts.length})
            </h2>
          </div>
          <div className="divide-y divide-viola-border">
            {relatedAlerts.map((alert) => (
              <Link
                key={alert.alert_id}
                href={`/alerts/${alert.alert_id}`}
                className="flex items-center gap-4 px-4 py-3 hover:bg-viola-border/20 transition-colors"
              >
                <span className="font-mono text-xs text-viola-muted w-36 shrink-0">{alert.alert_id}</span>
                <span className="text-sm text-viola-text flex-1 truncate">{alert.title}</span>
                <SeverityBadge severity={alert.severity} />
                <StatusBadge status={alert.status} />
                <span className="text-xs text-viola-muted shrink-0">
                  {formatRelative(alert.updated_at)}
                </span>
              </Link>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

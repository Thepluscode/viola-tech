import Link from "next/link";
import { SeverityBadge } from "@/components/shared/severity-badge";
import { StatusBadge } from "@/components/shared/status-badge";
import { RiskScoreBar } from "@/components/shared/risk-score-bar";
import { ConfidenceBar } from "@/components/shared/confidence-bar";
import { MitreTag } from "@/components/shared/mitre-tag";
import { LabelList } from "@/components/shared/label-list";
import { EntityList } from "@/components/shared/entity-list";
import { formatDate, formatRelative } from "@/lib/utils";
import type { Alert } from "@/types/api";
import { ArrowLeft, User, Calendar } from "lucide-react";

interface AlertDetailProps {
  alert: Alert;
}

export function AlertDetail({ alert }: AlertDetailProps) {
  return (
    <div className="max-w-5xl">
      <Link
        href="/alerts"
        className="inline-flex items-center gap-1.5 text-xs text-viola-muted hover:text-viola-accent mb-4 transition-colors"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Back to alerts
      </Link>

      {/* Header */}
      <div className="flex items-start justify-between flex-wrap gap-4 mb-6">
        <div>
          <div className="flex items-center gap-3 flex-wrap">
            <h1 className="font-mono text-base font-bold text-viola-muted">{alert.alert_id}</h1>
            <SeverityBadge severity={alert.severity} />
            <StatusBadge status={alert.status} />
          </div>
          <p className="text-lg font-semibold text-viola-text mt-1">{alert.title}</p>
          {alert.description && (
            <p className="text-sm text-viola-muted mt-1 max-w-2xl">{alert.description}</p>
          )}
        </div>
        <div className="text-right text-xs text-viola-muted space-y-1">
          <div className="flex items-center gap-1.5 justify-end">
            <Calendar className="h-3.5 w-3.5" />
            Created {formatDate(alert.created_at)}
          </div>
          <div className="flex items-center gap-1.5 justify-end">
            <Calendar className="h-3.5 w-3.5" />
            Updated {formatRelative(alert.updated_at)}
          </div>
          {alert.assigned_to && (
            <div className="flex items-center gap-1.5 justify-end">
              <User className="h-3.5 w-3.5" />
              {alert.assigned_to}
            </div>
          )}
        </div>
      </div>

      {/* Details grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
        {/* Risk & Confidence */}
        <div className="rounded-md border border-viola-border bg-viola-surface p-4 space-y-4">
          <h2 className="text-xs font-semibold text-viola-muted uppercase tracking-wider">
            Detection Scores
          </h2>
          <div>
            <p className="text-xs text-viola-muted mb-1.5">Risk Score</p>
            <RiskScoreBar score={alert.risk_score} />
          </div>
          <div>
            <p className="text-xs text-viola-muted mb-1.5">Confidence</p>
            {/* confidence is 0-1 float from backend */}
            <ConfidenceBar value={alert.confidence * 100} />
          </div>
        </div>

        {/* MITRE */}
        <div className="rounded-md border border-viola-border bg-viola-surface p-4 space-y-4">
          <h2 className="text-xs font-semibold text-viola-muted uppercase tracking-wider">
            MITRE ATT&CK
          </h2>
          <div className="space-y-2">
            {(alert.mitre_technique || alert.mitre_tactic) ? (
              <>
                <MitreTag
                  id={alert.mitre_technique ?? undefined}
                  tactic={alert.mitre_tactic ?? undefined}
                />
                <div className="text-xs text-viola-muted space-y-1">
                  {alert.mitre_tactic && (
                    <p>Tactic: <span className="text-viola-text">{alert.mitre_tactic}</span></p>
                  )}
                  {alert.mitre_technique && (
                    <p>Technique: <span className="font-mono text-viola-accent">{alert.mitre_technique}</span></p>
                  )}
                </div>
              </>
            ) : (
              <span className="text-xs text-viola-muted italic">No MITRE data</span>
            )}
          </div>
        </div>

        {/* Entities */}
        <div className="rounded-md border border-viola-border bg-viola-surface p-4 space-y-3">
          <h2 className="text-xs font-semibold text-viola-muted uppercase tracking-wider">
            Affected Entities
          </h2>
          <EntityList entities={alert.entity_ids} />
        </div>

        {/* Labels */}
        <div className="rounded-md border border-viola-border bg-viola-surface p-4 space-y-3">
          <h2 className="text-xs font-semibold text-viola-muted uppercase tracking-wider">Labels</h2>
          <LabelList labels={alert.labels} />
        </div>
      </div>

      {/* Closure reason if closed */}
      {alert.closure_reason && (
        <div className="rounded-md border border-viola-border bg-viola-surface p-4 mb-4">
          <h2 className="text-xs font-semibold text-viola-muted uppercase tracking-wider mb-2">
            Closure Reason
          </h2>
          <p className="text-sm text-viola-text">{alert.closure_reason}</p>
        </div>
      )}
    </div>
  );
}

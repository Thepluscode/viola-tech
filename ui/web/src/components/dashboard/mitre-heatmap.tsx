"use client";

import { cn } from "@/lib/utils";
import type { Alert, Incident } from "@/types/api";

// MITRE ATT&CK Enterprise tactics in kill-chain order.
const TACTICS = [
  "reconnaissance",
  "resource-development",
  "initial-access",
  "execution",
  "persistence",
  "privilege-escalation",
  "defense-evasion",
  "credential-access",
  "discovery",
  "lateral-movement",
  "collection",
  "command-and-control",
  "exfiltration",
  "impact",
] as const;

type Tactic = typeof TACTICS[number];

interface MitreHeatmapProps {
  incidents: Incident[];
  alerts: Alert[];
}

interface TacticStats {
  tactic: string;
  incidentCount: number;
  alertCount: number;
  criticalCount: number;
  maxRisk: number;
}

function buildTacticStats(incidents: Incident[], alerts: Alert[]): Map<string, TacticStats> {
  const map = new Map<string, TacticStats>();

  for (const tactic of TACTICS) {
    map.set(tactic, {
      tactic,
      incidentCount: 0,
      alertCount: 0,
      criticalCount: 0,
      maxRisk: 0,
    });
  }

  for (const inc of incidents) {
    const t = inc.mitre_tactic?.toLowerCase();
    if (!t) continue;
    const stats = map.get(t);
    if (!stats) continue;
    stats.incidentCount++;
    if (inc.severity === "critical") stats.criticalCount++;
    if ((inc.max_risk_score ?? 0) > stats.maxRisk) stats.maxRisk = inc.max_risk_score ?? 0;
  }

  for (const alert of alerts) {
    const t = alert.mitre_tactic?.toLowerCase();
    if (!t) continue;
    const stats = map.get(t);
    if (!stats) continue;
    stats.alertCount++;
    if (alert.severity === "critical") stats.criticalCount++;
    if ((alert.risk_score ?? 0) > stats.maxRisk) stats.maxRisk = alert.risk_score ?? 0;
  }

  return map;
}

function heatColor(maxRisk: number, total: number): string {
  if (total === 0) return "bg-viola-surface border-viola-border";
  if (maxRisk >= 80) return "bg-severity-critical-bg border-severity-critical";
  if (maxRisk >= 60) return "bg-severity-high-bg border-severity-high";
  if (maxRisk >= 40) return "bg-severity-medium-bg border-severity-medium";
  return "bg-severity-low-bg border-severity-low";
}

function tacticLabel(tactic: string): string {
  return tactic
    .split("-")
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(" ");
}

export function MitreHeatmap({ incidents, alerts }: MitreHeatmapProps) {
  const stats = buildTacticStats(incidents, alerts);

  return (
    <div className="rounded-md border border-viola-border bg-viola-surface p-4">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-sm font-semibold text-viola-text">MITRE ATT&CK Coverage</h2>
          <p className="text-xs text-viola-muted mt-0.5">Tactic activity heatmap</p>
        </div>
        <div className="flex gap-2 text-xs text-viola-muted">
          <span className="flex items-center gap-1">
            <span className="inline-block w-2.5 h-2.5 rounded-sm bg-severity-low-bg border border-severity-low" />
            Low
          </span>
          <span className="flex items-center gap-1">
            <span className="inline-block w-2.5 h-2.5 rounded-sm bg-severity-medium-bg border border-severity-medium" />
            Med
          </span>
          <span className="flex items-center gap-1">
            <span className="inline-block w-2.5 h-2.5 rounded-sm bg-severity-high-bg border border-severity-high" />
            High
          </span>
          <span className="flex items-center gap-1">
            <span className="inline-block w-2.5 h-2.5 rounded-sm bg-severity-critical-bg border border-severity-critical" />
            Critical
          </span>
        </div>
      </div>

      <div className="grid grid-cols-7 gap-1.5">
        {TACTICS.map((tactic) => {
          const s = stats.get(tactic)!;
          const total = s.incidentCount + s.alertCount;
          const color = heatColor(s.maxRisk, total);

          return (
            <div
              key={tactic}
              title={`${tacticLabel(tactic)}\nIncidents: ${s.incidentCount}\nAlerts: ${s.alertCount}\nMax risk: ${s.maxRisk.toFixed(0)}`}
              className={cn(
                "rounded-md border p-2 cursor-default transition-all",
                "hover:brightness-110 hover:scale-105",
                color,
                total === 0 && "opacity-40"
              )}
            >
              <p className="text-[9px] font-medium text-viola-text leading-tight truncate">
                {tacticLabel(tactic)}
              </p>
              {total > 0 ? (
                <p className="text-[11px] font-mono font-bold text-viola-text mt-1">{total}</p>
              ) : (
                <p className="text-[9px] text-viola-muted mt-1">—</p>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

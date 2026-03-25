import { KpiCard } from "./kpi-card";
import type { Incident, Alert } from "@/types/api";
import { ShieldAlert, Bell, AlertTriangle, CheckCircle2 } from "lucide-react";

interface KpiGridProps {
  incidents: { incidents: Incident[]; count: number };
  alerts: { alerts: Alert[]; count: number };
}

export function KpiGrid({ incidents, alerts }: KpiGridProps) {
  const openIncidents = incidents.incidents.filter((i) => i.status === "open").length;
  const criticalIncidents = incidents.incidents.filter((i) => i.severity === "critical").length;
  const openAlerts = alerts.alerts.filter((a) => a.status === "open").length;
  const ackdAlerts = alerts.alerts.filter((a) => a.status === "ack").length;

  // Build sparkline data from risk scores — sorted by updated_at ascending
  // so the chart reads left (oldest) → right (newest).
  const incidentRiskTrend = [...incidents.incidents]
    .sort((a, b) => new Date(a.updated_at).getTime() - new Date(b.updated_at).getTime())
    .map((i) => i.max_risk_score ?? 0);

  const alertRiskTrend = [...alerts.alerts]
    .sort((a, b) => new Date(a.updated_at).getTime() - new Date(b.updated_at).getTime())
    .map((a) => a.risk_score ?? 0);

  const criticalRiskTrend = [...incidents.incidents]
    .filter((i) => i.severity === "critical")
    .sort((a, b) => new Date(a.updated_at).getTime() - new Date(b.updated_at).getTime())
    .map((i) => i.max_risk_score ?? 0);

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      <KpiCard
        title="Open Incidents"
        value={openIncidents}
        subtitle={`${incidents.count} total`}
        icon={ShieldAlert}
        accentColor="text-severity-critical"
        sparklineValues={incidentRiskTrend}
        sparklineColor="#ef4444"
      />
      <KpiCard
        title="Critical Incidents"
        value={criticalIncidents}
        subtitle="Require immediate action"
        icon={AlertTriangle}
        accentColor="text-severity-critical"
        sparklineValues={criticalRiskTrend}
        sparklineColor="#ef4444"
      />
      <KpiCard
        title="Open Alerts"
        value={openAlerts}
        subtitle={`${ackdAlerts} acknowledged`}
        icon={Bell}
        accentColor="text-viola-accent"
        sparklineValues={alertRiskTrend}
        sparklineColor="#00d4ff"
      />
      <KpiCard
        title="Closed Today"
        value={
          incidents.incidents.filter(
            (i) =>
              i.status === "closed" &&
              i.updated_at &&
              new Date(i.updated_at).toDateString() === new Date().toDateString()
          ).length
        }
        subtitle="Incidents resolved"
        icon={CheckCircle2}
        accentColor="text-emerald-400"
      />
    </div>
  );
}

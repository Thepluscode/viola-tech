import { listIncidents, listAlerts } from "@/lib/api-client";
import { KpiGrid } from "@/components/dashboard/kpi-grid";
import { RecentActivity } from "@/components/dashboard/recent-activity";
import { MitreHeatmap } from "@/components/dashboard/mitre-heatmap";
import { LiveFeed } from "@/components/dashboard/live-feed";
import { Activity } from "lucide-react";
import { formatDate } from "@/lib/utils";

export const dynamic = "force-dynamic";

export default async function DashboardPage() {
  const [incidents, alerts] = await Promise.all([
    listIncidents({ status: "open" }),
    listAlerts({ status: "open" }),
  ]);

  return (
    <div className="px-6 py-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-bold text-viola-text">Security Dashboard</h1>
          <p className="text-xs text-viola-muted mt-0.5">
            Last updated: {formatDate(new Date().toISOString())}
          </p>
        </div>
        <div className="flex items-center gap-2 text-xs text-emerald-400 bg-emerald-950/40 border border-emerald-900 px-3 py-1.5 rounded-md">
          <Activity className="h-3.5 w-3.5" />
          Pipeline active
        </div>
      </div>

      <KpiGrid incidents={incidents} alerts={alerts} />

      <div className="mt-6">
        <MitreHeatmap
          incidents={incidents.incidents}
          alerts={alerts.alerts}
        />
      </div>

      <div className="mt-6 grid grid-cols-1 lg:grid-cols-2 gap-6">
        <RecentActivity incidents={incidents.incidents} alerts={alerts.alerts} />
        <LiveFeed />
      </div>
    </div>
  );
}

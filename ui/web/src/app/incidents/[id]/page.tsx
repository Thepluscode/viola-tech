import { notFound } from "next/navigation";
import { getIncident, listAlerts } from "@/lib/api-client";
import { IncidentDetail } from "@/components/incidents/incident-detail";
import { IncidentUpdateForm } from "@/components/incidents/incident-update-form";
import type { Metadata } from "next";

export const dynamic = "force-dynamic";

interface PageProps {
  params: { id: string };
}

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  return { title: params.id };
}

export default async function IncidentPage({ params }: PageProps) {
  const incident = await getIncident(params.id);
  if (!incident) notFound();

  // Fetch alerts for this incident — pass alert IDs from incident.alert_ids
  const alertsResult = await listAlerts({ incident_id: params.id });

  // Filter to only alerts whose alert_id is in incident.alert_ids
  const relatedAlerts = incident.alert_ids.length > 0
    ? alertsResult.alerts.filter((a) => incident.alert_ids.includes(a.alert_id))
    : alertsResult.alerts;

  return (
    <div className="px-6 py-6">
      <div className="grid grid-cols-1 xl:grid-cols-[1fr_300px] gap-6 max-w-6xl">
        <IncidentDetail incident={incident} relatedAlerts={relatedAlerts} />
        <div className="xl:pt-[60px]">
          <IncidentUpdateForm incident={incident} />
        </div>
      </div>
    </div>
  );
}

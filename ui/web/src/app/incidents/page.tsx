import { listIncidents } from "@/lib/api-client";
import { IncidentFilters } from "@/components/incidents/incident-filters";
import { IncidentsTable } from "@/components/incidents/incidents-table";
import { Pagination } from "@/components/shared/pagination";
import type { IncidentFilters as Filters, Severity, IncidentStatus } from "@/types/api";
import { PAGE_SIZE } from "@/types/api";
import type { Metadata } from "next";

export const dynamic = "force-dynamic";
export const metadata: Metadata = { title: "Incidents" };

interface PageProps {
  searchParams: { status?: string; severity?: string; page?: string };
}

export default async function IncidentsPage({ searchParams }: PageProps) {
  const page = Number(searchParams.page ?? 1);
  const filters: Filters & { page: number } = {
    page,
    ...(searchParams.status ? { status: searchParams.status as IncidentStatus } : {}),
    ...(searchParams.severity ? { severity: searchParams.severity as Severity } : {}),
  };

  const result = await listIncidents(filters);
  const hasNextPage = result.incidents.length === PAGE_SIZE;

  return (
    <div className="px-6 py-6">
      <div className="mb-6">
        <h1 className="text-xl font-bold text-viola-text">Incidents</h1>
        <p className="text-xs text-viola-muted mt-0.5">
          {result.count} incident{result.count !== 1 ? "s" : ""} found
        </p>
      </div>

      <div className="mb-4">
        <IncidentFilters />
      </div>

      <IncidentsTable incidents={result.incidents} />

      <div className="mt-4 flex justify-end">
        <Pagination page={page} hasNextPage={hasNextPage} />
      </div>
    </div>
  );
}

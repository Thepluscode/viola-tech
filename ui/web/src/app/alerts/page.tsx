import { listAlerts } from "@/lib/api-client";
import { AlertFilters } from "@/components/alerts/alert-filters";
import { AlertsTable } from "@/components/alerts/alerts-table";
import { Pagination } from "@/components/shared/pagination";
import type { AlertFilters as Filters, Severity, AlertStatus } from "@/types/api";
import { PAGE_SIZE } from "@/types/api";
import type { Metadata } from "next";

export const dynamic = "force-dynamic";
export const metadata: Metadata = { title: "Alerts" };

interface PageProps {
  searchParams: { status?: string; severity?: string; page?: string };
}

export default async function AlertsPage({ searchParams }: PageProps) {
  const page = Number(searchParams.page ?? 1);
  const filters: Filters & { page: number } = {
    page,
    ...(searchParams.status ? { status: searchParams.status as AlertStatus } : {}),
    ...(searchParams.severity ? { severity: searchParams.severity as Severity } : {}),
  };

  const result = await listAlerts(filters);
  const hasNextPage = result.alerts.length === PAGE_SIZE;

  return (
    <div className="px-6 py-6">
      <div className="mb-6">
        <h1 className="text-xl font-bold text-viola-text">Alerts</h1>
        <p className="text-xs text-viola-muted mt-0.5">
          {result.count} alert{result.count !== 1 ? "s" : ""} found
        </p>
      </div>

      <div className="mb-4">
        <AlertFilters />
      </div>

      <AlertsTable alerts={result.alerts} />

      <div className="mt-4 flex justify-end">
        <Pagination page={page} hasNextPage={hasNextPage} />
      </div>
    </div>
  );
}

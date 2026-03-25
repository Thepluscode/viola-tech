import { KpiGridSkeleton } from "@/components/shared/loading-skeleton";
import { TableSkeleton } from "@/components/shared/loading-skeleton";

export default function DashboardLoading() {
  return (
    <div className="px-6 py-6">
      <div className="mb-6">
        <div className="h-6 w-48 rounded-md bg-viola-surface animate-pulse" />
        <div className="h-3 w-32 rounded-md bg-viola-surface animate-pulse mt-2" />
      </div>
      <KpiGridSkeleton />
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-6">
        <TableSkeleton rows={5} />
        <TableSkeleton rows={5} />
      </div>
    </div>
  );
}

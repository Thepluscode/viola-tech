import { TableSkeleton } from "@/components/shared/loading-skeleton";

export default function AlertsLoading() {
  return (
    <div className="px-6 py-6">
      <div className="mb-6">
        <div className="h-6 w-24 rounded-md bg-viola-surface animate-pulse" />
        <div className="h-3 w-20 rounded-md bg-viola-surface animate-pulse mt-2" />
      </div>
      <div className="h-8 w-64 rounded-md bg-viola-surface animate-pulse mb-4" />
      <TableSkeleton rows={8} />
    </div>
  );
}

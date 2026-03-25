import { Skeleton } from "@/components/shared/loading-skeleton";

export default function IncidentDetailLoading() {
  return (
    <div className="px-6 py-6">
      <Skeleton className="h-4 w-24 mb-4" />
      <div className="flex gap-3 mb-4">
        <Skeleton className="h-6 w-32" />
        <Skeleton className="h-6 w-20" />
        <Skeleton className="h-6 w-24" />
      </div>
      <Skeleton className="h-4 w-2/3 mb-6" />
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {[1, 2, 3, 4].map((i) => (
          <Skeleton key={i} className="h-40" />
        ))}
      </div>
    </div>
  );
}

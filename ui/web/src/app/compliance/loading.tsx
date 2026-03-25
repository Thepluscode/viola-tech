import { LoadingSkeleton } from "@/components/shared/loading-skeleton";

export default function ComplianceLoading() {
  return (
    <div className="px-6 py-6">
      <LoadingSkeleton className="h-8 w-64 mb-2" />
      <LoadingSkeleton className="h-4 w-48 mb-6" />

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <LoadingSkeleton key={i} className="h-36" />
        ))}
      </div>

      <LoadingSkeleton className="h-96 mt-6" />
    </div>
  );
}

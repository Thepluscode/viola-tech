import { LoadingSkeleton } from "@/components/shared/loading-skeleton";

export default function GraphLoading() {
  return (
    <div className="px-6 py-6">
      <LoadingSkeleton className="h-8 w-48 mb-2" />
      <LoadingSkeleton className="h-4 w-64 mb-6" />
      <LoadingSkeleton className="h-12 mb-4" />
      <LoadingSkeleton className="h-[600px]" />
    </div>
  );
}

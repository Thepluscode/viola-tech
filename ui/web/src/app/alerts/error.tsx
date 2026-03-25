"use client";

import { ErrorState } from "@/components/shared/error-state";

export default function AlertsError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="px-6 py-6">
      <ErrorState
        title="Failed to load alerts"
        message={error.message || "An unexpected error occurred."}
      />
      <div className="flex justify-center mt-4">
        <button
          onClick={reset}
          className="text-xs text-viola-accent hover:underline px-3 py-1.5 rounded-md border border-viola-accent/30 bg-viola-accent/10 hover:bg-viola-accent/20 transition-colors"
        >
          Try again
        </button>
      </div>
    </div>
  );
}

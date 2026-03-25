import Link from "next/link";
import { BellOff } from "lucide-react";

export default function AlertNotFound() {
  return (
    <div className="flex flex-col items-center justify-center min-h-[60vh] px-6 gap-4">
      <div className="flex items-center justify-center w-12 h-12 rounded-md bg-viola-surface border border-viola-border">
        <BellOff className="h-6 w-6 text-viola-muted" />
      </div>
      <div className="text-center">
        <p className="text-sm font-semibold text-viola-text">Alert not found</p>
        <p className="text-xs text-viola-muted mt-1">
          The alert you&apos;re looking for doesn&apos;t exist or has been removed.
        </p>
      </div>
      <Link href="/alerts" className="text-xs text-viola-accent hover:underline">
        ← Back to alerts
      </Link>
    </div>
  );
}

import Link from "next/link";
import { ShieldOff } from "lucide-react";

export default function IncidentNotFound() {
  return (
    <div className="flex flex-col items-center justify-center min-h-[60vh] px-6 gap-4">
      <div className="flex items-center justify-center w-12 h-12 rounded-md bg-viola-surface border border-viola-border">
        <ShieldOff className="h-6 w-6 text-viola-muted" />
      </div>
      <div className="text-center">
        <p className="text-sm font-semibold text-viola-text">Incident not found</p>
        <p className="text-xs text-viola-muted mt-1">
          The incident you&apos;re looking for doesn&apos;t exist or has been removed.
        </p>
      </div>
      <Link
        href="/incidents"
        className="text-xs text-viola-accent hover:underline"
      >
        ← Back to incidents
      </Link>
    </div>
  );
}

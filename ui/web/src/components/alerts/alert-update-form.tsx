"use client";

import { useState, useTransition } from "react";
import { useRouter } from "next/navigation";
import { updateAlert } from "@/lib/api-client";
import type { Alert, AlertStatus } from "@/types/api";
import { Check, Loader2 } from "lucide-react";

interface AlertUpdateFormProps {
  alert: Alert;
}

const statuses: { value: AlertStatus; label: string }[] = [
  { value: "open", label: "Open" },
  { value: "ack", label: "Acknowledged" },
  { value: "closed", label: "Closed" },
];

export function AlertUpdateForm({ alert }: AlertUpdateFormProps) {
  const router = useRouter();
  const [isPending, startTransition] = useTransition();
  const [status, setStatus] = useState<AlertStatus>(alert.status);
  const [assignedTo, setAssignedTo] = useState(alert.assigned_to ?? "");
  const [closureReason, setClosureReason] = useState(alert.closure_reason ?? "");
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSaved(false);

    startTransition(async () => {
      try {
        await updateAlert(alert.alert_id, {
          status,
          assigned_to: assignedTo || null,
          closure_reason: closureReason || null,
        });
        setSaved(true);
        router.refresh();
        setTimeout(() => setSaved(false), 2000);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Update failed");
      }
    });
  }

  return (
    <div className="rounded-md border border-viola-border bg-viola-surface p-4">
      <h2 className="text-xs font-semibold text-viola-muted uppercase tracking-wider mb-4">
        Update Alert
      </h2>
      <form onSubmit={handleSubmit} className="space-y-3">
        <div>
          <label className="block text-xs text-viola-muted mb-1.5">Status</label>
          <select
            value={status}
            onChange={(e) => setStatus(e.target.value as AlertStatus)}
            className="w-full text-sm bg-viola-bg border border-viola-border text-viola-text rounded-md px-3 py-2 focus:outline-none focus:border-viola-accent"
          >
            {statuses.map((s) => (
              <option key={s.value} value={s.value}>{s.label}</option>
            ))}
          </select>
        </div>

        <div>
          <label className="block text-xs text-viola-muted mb-1.5">Assigned To</label>
          <input
            type="text"
            value={assignedTo}
            onChange={(e) => setAssignedTo(e.target.value)}
            placeholder="analyst@viola.corp"
            className="w-full text-sm bg-viola-bg border border-viola-border text-viola-text rounded-md px-3 py-2 focus:outline-none focus:border-viola-accent placeholder:text-viola-border"
          />
        </div>

        {status === "closed" && (
          <div>
            <label className="block text-xs text-viola-muted mb-1.5">Closure Reason</label>
            <input
              type="text"
              value={closureReason}
              onChange={(e) => setClosureReason(e.target.value)}
              placeholder="False positive, resolved, etc."
              className="w-full text-sm bg-viola-bg border border-viola-border text-viola-text rounded-md px-3 py-2 focus:outline-none focus:border-viola-accent placeholder:text-viola-border"
            />
          </div>
        )}

        {error && <p className="text-xs text-severity-critical">{error}</p>}

        <button
          type="submit"
          disabled={isPending}
          className="flex items-center gap-2 px-4 py-2 rounded-md bg-viola-accent/10 border border-viola-accent/30 text-viola-accent text-sm hover:bg-viola-accent/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : saved ? <Check className="h-3.5 w-3.5" /> : null}
          {isPending ? "Saving..." : saved ? "Saved!" : "Save Changes"}
        </button>
      </form>
    </div>
  );
}

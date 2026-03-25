import { cn } from "@/lib/utils";
import type { IncidentStatus, AlertStatus } from "@/types/api";

type Status = IncidentStatus | AlertStatus;

interface StatusBadgeProps {
  status: Status;
  className?: string;
}

const statusConfig: Record<Status, { label: string; classes: string }> = {
  open: {
    label: "Open",
    classes: "bg-status-open-bg text-status-open border border-status-open-border",
  },
  ack: {
    label: "Acknowledged",
    classes: "bg-status-ack-bg text-status-ack border border-status-ack-border",
  },
  closed: {
    label: "Closed",
    classes: "bg-status-closed-bg text-status-closed border border-status-closed-border",
  },
};

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const config = statusConfig[status] ?? statusConfig.open;
  return (
    <span
      className={cn(
        "inline-flex items-center px-2 py-0.5 text-xs font-semibold uppercase tracking-wider rounded-md",
        config.classes,
        className
      )}
    >
      {config.label}
    </span>
  );
}

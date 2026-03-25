"use client";

import { cn } from "@/lib/utils";
import { formatRelative } from "@/lib/utils";
import type { AuditEvent } from "@/types/compliance";
import { FRAMEWORK_LABELS } from "@/types/compliance";
import { CheckCircle2, XCircle, Info } from "lucide-react";

interface AuditTrailProps {
  events: AuditEvent[];
}

const RESULT_CONFIG = {
  pass: { icon: CheckCircle2, color: "text-emerald-400", bg: "bg-emerald-950/40" },
  fail: { icon: XCircle, color: "text-red-400", bg: "bg-red-950/40" },
  info: { icon: Info, color: "text-viola-accent", bg: "bg-viola-accent/10" },
} as const;

export function AuditTrail({ events }: AuditTrailProps) {
  return (
    <div className="rounded-md border border-viola-border bg-viola-surface">
      <div className="p-3 border-b border-viola-border">
        <h3 className="text-sm font-semibold text-viola-text">Audit Trail</h3>
        <p className="text-[10px] text-viola-muted mt-0.5">Recent compliance events</p>
      </div>

      <div className="divide-y divide-viola-border/50">
        {events.map((event) => {
          const cfg = RESULT_CONFIG[event.result];
          const Icon = cfg.icon;

          return (
            <div
              key={event.event_id}
              className="flex items-start gap-3 px-3 py-2.5 hover:bg-viola-border/20 transition-colors"
            >
              <div className={cn("p-1 rounded", cfg.bg)}>
                <Icon className={cn("h-3.5 w-3.5", cfg.color)} />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="text-xs text-viola-text font-medium truncate">
                    {event.action.replace(/_/g, " ")}
                  </span>
                  <span className="text-[10px] text-viola-muted">
                    {FRAMEWORK_LABELS[event.framework]}
                  </span>
                </div>
                <p className="text-[10px] text-viola-muted mt-0.5 truncate">
                  {event.resource}
                </p>
                <div className="flex items-center gap-2 mt-0.5">
                  <span className="text-[10px] text-viola-muted">{event.actor}</span>
                  <span className="text-[10px] text-viola-muted">
                    {formatRelative(event.timestamp)}
                  </span>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

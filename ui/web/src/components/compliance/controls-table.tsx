"use client";

import { cn } from "@/lib/utils";
import { formatDate } from "@/lib/utils";
import type { ComplianceControl, ControlStatus, ComplianceFramework } from "@/types/compliance";
import { CONTROL_STATUS_LABELS, FRAMEWORK_LABELS } from "@/types/compliance";
import { useState } from "react";

interface ControlsTableProps {
  controls: ComplianceControl[];
}

const STATUS_COLORS: Record<ControlStatus, string> = {
  passing: "bg-emerald-950/60 text-emerald-400 border-emerald-900",
  failing: "bg-red-950/60 text-red-400 border-red-900",
  partial: "bg-yellow-950/60 text-yellow-400 border-yellow-900",
  "not-assessed": "bg-viola-border/50 text-viola-muted border-viola-border",
};

export function ControlsTable({ controls }: ControlsTableProps) {
  const [frameworkFilter, setFrameworkFilter] = useState<ComplianceFramework | "all">("all");
  const [statusFilter, setStatusFilter] = useState<ControlStatus | "all">("all");

  const filtered = controls.filter((c) => {
    if (frameworkFilter !== "all" && c.framework !== frameworkFilter) return false;
    if (statusFilter !== "all" && c.status !== statusFilter) return false;
    return true;
  });

  return (
    <div className="rounded-md border border-viola-border bg-viola-surface">
      <div className="flex items-center gap-3 p-3 border-b border-viola-border">
        <h3 className="text-sm font-semibold text-viola-text flex-1">Controls</h3>
        <select
          value={frameworkFilter}
          onChange={(e) => setFrameworkFilter(e.target.value as ComplianceFramework | "all")}
          className="text-xs bg-viola-bg border border-viola-border text-viola-text rounded px-2 py-1"
        >
          <option value="all">All Frameworks</option>
          {(Object.keys(FRAMEWORK_LABELS) as ComplianceFramework[]).map((f) => (
            <option key={f} value={f}>{FRAMEWORK_LABELS[f]}</option>
          ))}
        </select>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value as ControlStatus | "all")}
          className="text-xs bg-viola-bg border border-viola-border text-viola-text rounded px-2 py-1"
        >
          <option value="all">All Statuses</option>
          {(Object.keys(CONTROL_STATUS_LABELS) as ControlStatus[]).map((s) => (
            <option key={s} value={s}>{CONTROL_STATUS_LABELS[s]}</option>
          ))}
        </select>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-xs">
          <thead>
            <tr className="border-b border-viola-border text-viola-muted">
              <th className="text-left px-3 py-2 font-medium">Control ID</th>
              <th className="text-left px-3 py-2 font-medium">Title</th>
              <th className="text-left px-3 py-2 font-medium">Framework</th>
              <th className="text-left px-3 py-2 font-medium">Status</th>
              <th className="text-left px-3 py-2 font-medium">Evidence</th>
              <th className="text-left px-3 py-2 font-medium">Last Assessed</th>
              <th className="text-left px-3 py-2 font-medium">Owner</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((control) => (
              <tr
                key={`${control.framework}-${control.control_id}`}
                className="border-b border-viola-border/50 hover:bg-viola-border/20 transition-colors"
              >
                <td className="px-3 py-2 font-mono text-viola-accent">
                  {control.control_id}
                </td>
                <td className="px-3 py-2 text-viola-text max-w-xs">
                  <div className="truncate">{control.title}</div>
                  {control.notes && (
                    <div className="text-[10px] text-yellow-400/80 mt-0.5 truncate">
                      {control.notes}
                    </div>
                  )}
                </td>
                <td className="px-3 py-2 text-viola-muted">
                  {FRAMEWORK_LABELS[control.framework]}
                </td>
                <td className="px-3 py-2">
                  <span
                    className={cn(
                      "inline-flex px-2 py-0.5 rounded text-[10px] font-medium border",
                      STATUS_COLORS[control.status]
                    )}
                  >
                    {CONTROL_STATUS_LABELS[control.status]}
                  </span>
                </td>
                <td className="px-3 py-2 font-mono text-viola-muted">
                  {control.evidence_count}
                </td>
                <td className="px-3 py-2 text-viola-muted">
                  {formatDate(control.last_assessed)}
                </td>
                <td className="px-3 py-2 text-viola-muted">
                  {control.owner ?? "—"}
                </td>
              </tr>
            ))}
            {filtered.length === 0 && (
              <tr>
                <td colSpan={7} className="px-3 py-8 text-center text-viola-muted">
                  No controls match the selected filters.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

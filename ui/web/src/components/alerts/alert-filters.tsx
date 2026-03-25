"use client";

import { useRouter, usePathname, useSearchParams } from "next/navigation";
import { Filter, X } from "lucide-react";
import type { Severity, AlertStatus } from "@/types/api";
import { STATUS_LABELS } from "@/types/api";

const severities: Severity[] = ["critical", "high", "medium", "low"];
const statuses: AlertStatus[] = ["open", "ack", "closed"];

export function AlertFilters() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const currentStatus = searchParams.get("status") ?? "";
  const currentSeverity = searchParams.get("severity") ?? "";

  function updateFilter(key: string, value: string) {
    const params = new URLSearchParams(searchParams.toString());
    if (value) {
      params.set(key, value);
    } else {
      params.delete(key);
    }
    params.delete("page");
    router.push(`${pathname}?${params.toString()}`);
  }

  function clearAll() {
    router.push(pathname);
  }

  const hasFilters = currentStatus || currentSeverity;

  return (
    <div className="flex items-center gap-3 flex-wrap">
      <div className="flex items-center gap-1.5 text-xs text-viola-muted">
        <Filter className="h-3.5 w-3.5" />
        Filters:
      </div>

      <select
        value={currentStatus}
        onChange={(e) => updateFilter("status", e.target.value)}
        className="text-xs bg-viola-surface border border-viola-border text-viola-text rounded-md px-2.5 py-1.5 focus:outline-none focus:border-viola-accent cursor-pointer"
      >
        <option value="">All Statuses</option>
        {statuses.map((s) => (
          <option key={s} value={s}>{STATUS_LABELS[s]}</option>
        ))}
      </select>

      <select
        value={currentSeverity}
        onChange={(e) => updateFilter("severity", e.target.value)}
        className="text-xs bg-viola-surface border border-viola-border text-viola-text rounded-md px-2.5 py-1.5 focus:outline-none focus:border-viola-accent cursor-pointer"
      >
        <option value="">All Severities</option>
        {severities.map((s) => (
          <option key={s} value={s}>
            {s.charAt(0).toUpperCase() + s.slice(1)}
          </option>
        ))}
      </select>

      {hasFilters && (
        <button
          onClick={clearAll}
          className="flex items-center gap-1 text-xs text-viola-muted hover:text-viola-text px-2 py-1.5 rounded-md hover:bg-viola-border/50 transition-colors"
        >
          <X className="h-3.5 w-3.5" />
          Clear
        </button>
      )}
    </div>
  );
}

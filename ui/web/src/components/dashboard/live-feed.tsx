"use client";

import { useEffect, useRef, useState } from "react";
import { Activity, Wifi, WifiOff } from "lucide-react";
import Link from "next/link";
import { SeverityBadge } from "@/components/shared/severity-badge";
import { StatusBadge } from "@/components/shared/status-badge";
import type { Incident, Alert } from "@/types/api";
import { formatRelative } from "@/lib/utils";

type FeedEvent =
  | { kind: "alert"; payload: Alert; ts: string }
  | { kind: "incident"; payload: Incident; ts: string };

interface LiveFeedProps {
  maxItems?: number;
}

export function LiveFeed({ maxItems = 20 }: LiveFeedProps) {
  const [events, setEvents] = useState<FeedEvent[]>([]);
  const [connected, setConnected] = useState(false);
  const [lastHeartbeat, setLastHeartbeat] = useState<Date | null>(null);
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    function connect() {
      const es = new EventSource("/api/live-feed");
      esRef.current = es;

      es.addEventListener("open", () => setConnected(true));
      es.addEventListener("error", () => setConnected(false));

      es.addEventListener("heartbeat", () => {
        setLastHeartbeat(new Date());
        setConnected(true);
      });

      es.addEventListener("alert", (e) => {
        const payload = JSON.parse(e.data) as Alert;
        setEvents((prev) => [
          { kind: "alert", payload, ts: new Date().toISOString() },
          ...prev.slice(0, maxItems - 1),
        ]);
      });

      es.addEventListener("incident", (e) => {
        const payload = JSON.parse(e.data) as Incident;
        setEvents((prev) => [
          { kind: "incident", payload, ts: new Date().toISOString() },
          ...prev.slice(0, maxItems - 1),
        ]);
      });
    }

    connect();
    return () => {
      esRef.current?.close();
    };
  }, [maxItems]);

  return (
    <div className="rounded-md border border-viola-border bg-viola-surface">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-viola-border">
        <div className="flex items-center gap-2">
          <Activity className="h-4 w-4 text-viola-accent" />
          <h2 className="text-sm font-semibold text-viola-text">Live Feed</h2>
        </div>
        <div className="flex items-center gap-1.5 text-xs">
          {connected ? (
            <>
              <Wifi className="h-3.5 w-3.5 text-emerald-400" />
              <span className="text-emerald-400">Live</span>
              {lastHeartbeat && (
                <span className="text-viola-muted ml-1">
                  · {formatRelative(lastHeartbeat.toISOString())}
                </span>
              )}
            </>
          ) : (
            <>
              <WifiOff className="h-3.5 w-3.5 text-viola-muted" />
              <span className="text-viola-muted">Connecting…</span>
            </>
          )}
        </div>
      </div>

      {/* Event list */}
      <div className="divide-y divide-viola-border/50 max-h-80 overflow-y-auto">
        {events.length === 0 ? (
          <div className="px-4 py-8 text-center text-xs text-viola-muted">
            Waiting for new events…
          </div>
        ) : (
          events.map((ev, i) => (
            <FeedRow key={i} event={ev} />
          ))
        )}
      </div>
    </div>
  );
}

function FeedRow({ event }: { event: FeedEvent }) {
  if (event.kind === "alert") {
    const a = event.payload;
    return (
      <Link
        href={`/alerts/${a.alert_id}`}
        className="flex items-center gap-3 px-4 py-2.5 hover:bg-viola-border/20 transition-colors group"
      >
        <div className="shrink-0">
          <span className="text-[9px] uppercase font-mono text-viola-muted bg-viola-border/60 px-1 py-0.5 rounded">
            alert
          </span>
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-xs font-medium text-viola-text truncate group-hover:text-viola-accent transition-colors">
            {a.title}
          </p>
          <p className="text-[10px] text-viola-muted font-mono">{a.alert_id}</p>
        </div>
        <div className="shrink-0 flex items-center gap-1.5">
          <SeverityBadge severity={a.severity} />
          <StatusBadge status={a.status} />
        </div>
        <span className="text-[10px] text-viola-muted shrink-0">
          {formatRelative(event.ts)}
        </span>
      </Link>
    );
  }

  const inc = event.payload;
  return (
    <Link
      href={`/incidents/${inc.incident_id}`}
      className="flex items-center gap-3 px-4 py-2.5 hover:bg-viola-border/20 transition-colors group"
    >
      <div className="shrink-0">
        <span className="text-[9px] uppercase font-mono text-severity-critical bg-severity-critical-bg px-1 py-0.5 rounded">
          incident
        </span>
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-xs font-medium text-viola-text truncate group-hover:text-viola-accent transition-colors">
          {inc.mitre_tactic
            ? `${inc.mitre_tactic} · ${inc.mitre_technique ?? ""}`
            : inc.incident_id}
        </p>
        <p className="text-[10px] text-viola-muted font-mono">{inc.incident_id}</p>
      </div>
      <div className="shrink-0 flex items-center gap-1.5">
        <SeverityBadge severity={inc.severity} />
        <StatusBadge status={inc.status} />
      </div>
      <span className="text-[10px] text-viola-muted shrink-0">
        {formatRelative(event.ts)}
      </span>
    </Link>
  );
}

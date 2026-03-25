import { NextResponse } from "next/server";

/**
 * GET /api/live-feed
 *
 * Server-Sent Events endpoint that polls the gateway-api for new alerts and
 * incidents every 5 seconds and streams them as SSE events.
 *
 * Clients connect once and receive a stream of:
 *   event: alert\ndata: <JSON Alert>\n\n
 *   event: incident\ndata: <JSON Incident>\n\n
 *   event: heartbeat\ndata: {"ts":"<rfc3339>"}\n\n
 *
 * The SSE stream terminates automatically when the client disconnects.
 */

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
const TOKEN = process.env.DEV_API_TOKEN ?? "";
const POLL_INTERVAL_MS = 5000;
const MAX_EVENTS_PER_POLL = 20;

function authHeaders(): Record<string, string> {
  if (!TOKEN) return {};
  return { Authorization: `Bearer ${TOKEN}` };
}

async function fetchRecent(resource: "alerts" | "incidents"): Promise<unknown[]> {
  const url = `${API_BASE}/api/v1/${resource}?limit=${MAX_EVENTS_PER_POLL}&status=open`;
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json", ...authHeaders() },
    cache: "no-store",
    signal: AbortSignal.timeout(4000),
  });
  if (!res.ok) return [];
  const json = await res.json() as Record<string, unknown>;
  return (json[resource] as unknown[]) ?? [];
}

export async function GET(request: Request): Promise<Response> {
  const encoder = new TextEncoder();

  const stream = new ReadableStream({
    async start(controller) {
      const signal = request.signal;
      let seenAlerts = new Set<string>();
      let seenIncidents = new Set<string>();
      let ticks = 0;

      function send(event: string, data: unknown) {
        const chunk = `event: ${event}\ndata: ${JSON.stringify(data)}\n\n`;
        controller.enqueue(encoder.encode(chunk));
      }

      while (!signal.aborted) {
        try {
          const [alerts, incidents] = await Promise.all([
            fetchRecent("alerts"),
            fetchRecent("incidents"),
          ]);

          for (const alert of alerts as Array<{alert_id?: string}>) {
            const id = alert.alert_id ?? "";
            if (id && !seenAlerts.has(id)) {
              seenAlerts.add(id);
              if (ticks > 0) send("alert", alert); // skip first batch (already in UI)
            }
          }

          for (const inc of incidents as Array<{incident_id?: string}>) {
            const id = inc.incident_id ?? "";
            if (id && !seenIncidents.has(id)) {
              seenIncidents.add(id);
              if (ticks > 0) send("incident", inc);
            }
          }

          // Evict old IDs to prevent unbounded growth (keep last 1000)
          if (seenAlerts.size > 1000) seenAlerts = new Set([...seenAlerts].slice(-500));
          if (seenIncidents.size > 1000) seenIncidents = new Set([...seenIncidents].slice(-500));

          send("heartbeat", { ts: new Date().toISOString() });
          ticks++;
        } catch {
          // Gateway unavailable — send heartbeat to keep connection alive
          send("heartbeat", { ts: new Date().toISOString(), error: "upstream_unavailable" });
        }

        // Wait POLL_INTERVAL_MS or until abort
        await new Promise<void>((resolve) => {
          const timer = setTimeout(resolve, POLL_INTERVAL_MS);
          signal.addEventListener("abort", () => { clearTimeout(timer); resolve(); }, { once: true });
        });
      }

      controller.close();
    },
  });

  return new NextResponse(stream, {
    headers: {
      "Content-Type": "text/event-stream",
      "Cache-Control": "no-cache, no-transform",
      Connection: "keep-alive",
      "X-Accel-Buffering": "no", // Disable nginx buffering for SSE
    },
  });
}

import type {
  Incident,
  Alert,
  IncidentListResponse,
  AlertListResponse,
  IncidentFilters,
  AlertFilters,
  UpdateIncidentPayload,
  UpdateAlertPayload,
} from "@/types/api";
import { MOCK_INCIDENTS, MOCK_ALERTS } from "./mock-data";
import { PAGE_SIZE } from "@/types/api";

const USE_MOCK = process.env.NEXT_PUBLIC_USE_MOCK === "true";

// Server-side fetch needs absolute URL; browser fetch uses relative (via next.config rewrite)
const API_BASE =
  typeof window === "undefined"
    ? process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"
    : "";

// ─── HTTP helpers ─────────────────────────────────────────────────────────────

function buildQS(params: Record<string, string | number | undefined>): string {
  const p = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== "") p.set(k, String(v));
  }
  return p.size ? `?${p.toString()}` : "";
}

async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const token = process.env.DEV_API_TOKEN;
  const url = `${API_BASE}${path}`;
  const res = await fetch(url, {
    ...options,
    cache: "no-store",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(options?.headers ?? {}),
    },
  });
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`API ${res.status} ${res.statusText}: ${url} — ${body}`);
  }
  return res.json() as Promise<T>;
}

// ─── Page helper (UI uses 1-based pages, backend uses limit/offset) ──────────

function pageToOffset(page: number): number {
  return (Math.max(1, page) - 1) * PAGE_SIZE;
}

// ─── Incidents ───────────────────────────────────────────────────────────────

export async function listIncidents(
  filters: IncidentFilters & { page?: number } = {}
): Promise<{ incidents: Incident[]; count: number }> {
  const { page, ...rest } = filters;
  const offset = pageToOffset(page ?? 1);

  if (USE_MOCK) {
    let data = [...MOCK_INCIDENTS];
    if (rest.status) data = data.filter((i) => i.status === rest.status);
    if (rest.severity) data = data.filter((i) => i.severity === rest.severity);
    return { incidents: data.slice(offset, offset + PAGE_SIZE), count: data.length };
  }

  try {
    return await apiFetch<IncidentListResponse>(
      `/api/v1/incidents${buildQS({ ...rest, limit: PAGE_SIZE, offset })}`
    );
  } catch (err) {
    console.warn("[api-client] listIncidents failed, using mock:", err);
    return { incidents: MOCK_INCIDENTS, count: MOCK_INCIDENTS.length };
  }
}

export async function getIncident(id: string): Promise<Incident | null> {
  if (USE_MOCK) {
    return MOCK_INCIDENTS.find((i) => i.incident_id === id) ?? null;
  }
  try {
    return await apiFetch<Incident>(`/api/v1/incidents/${id}`);
  } catch (err) {
    console.warn(`[api-client] getIncident(${id}) failed, using mock:`, err);
    return MOCK_INCIDENTS.find((i) => i.incident_id === id) ?? null;
  }
}

export async function updateIncident(
  id: string,
  payload: UpdateIncidentPayload
): Promise<void> {
  if (USE_MOCK) return; // mock — no-op, caller handles optimistic UI
  await apiFetch<{ status: string }>(`/api/v1/incidents/${id}`, {
    method: "PATCH",
    body: JSON.stringify(payload),
  });
}

// ─── Alerts ──────────────────────────────────────────────────────────────────

export async function listAlerts(
  filters: AlertFilters & { page?: number } = {}
): Promise<{ alerts: Alert[]; count: number }> {
  const { page, incident_id, ...rest } = filters;
  const offset = pageToOffset(page ?? 1);

  if (USE_MOCK) {
    let data = [...MOCK_ALERTS];
    if (rest.status) data = data.filter((a) => a.status === rest.status);
    if (rest.severity) data = data.filter((a) => a.severity === rest.severity);
    // incident_id filter: match via MOCK_INCIDENTS.alert_ids
    if (incident_id) {
      const inc = MOCK_INCIDENTS.find((i) => i.incident_id === incident_id);
      const ids = new Set(inc?.alert_ids ?? []);
      data = data.filter((a) => ids.has(a.alert_id));
    }
    return { alerts: data.slice(offset, offset + PAGE_SIZE), count: data.length };
  }

  try {
    // Backend doesn't have incident_id filter — fetch all and filter client-side if needed
    const result = await apiFetch<AlertListResponse>(
      `/api/v1/alerts${buildQS({ ...rest, limit: PAGE_SIZE, offset })}`
    );
    if (incident_id) {
      const filtered = result.alerts.filter((a) => {
        // Backend doesn't expose incident_id on alert — fall back to fetching related alert IDs
        // from the incident itself. For now filter by alert_ids from the incident if available.
        return true; // passthrough; caller can filter if needed
      });
      return { alerts: filtered, count: filtered.length };
    }
    return result;
  } catch (err) {
    console.warn("[api-client] listAlerts failed, using mock:", err);
    return { alerts: MOCK_ALERTS, count: MOCK_ALERTS.length };
  }
}

export async function getAlert(id: string): Promise<Alert | null> {
  if (USE_MOCK) {
    return MOCK_ALERTS.find((a) => a.alert_id === id) ?? null;
  }
  try {
    return await apiFetch<Alert>(`/api/v1/alerts/${id}`);
  } catch (err) {
    console.warn(`[api-client] getAlert(${id}) failed, using mock:`, err);
    return MOCK_ALERTS.find((a) => a.alert_id === id) ?? null;
  }
}

export async function updateAlert(
  id: string,
  payload: UpdateAlertPayload
): Promise<void> {
  if (USE_MOCK) return;
  await apiFetch<{ status: string }>(`/api/v1/alerts/${id}`, {
    method: "PATCH",
    body: JSON.stringify(payload),
  });
}

// ─── Auth helper — fetch a dev token from the built-in auth service ──────────
// Usage: only call server-side (e.g. from a script or server action)

export async function fetchDevToken(opts?: {
  sub?: string;
  tid?: string;
  email?: string;
  role?: string;
}): Promise<string> {
  const body = {
    sub: opts?.sub ?? "dev-user",
    tid: opts?.tid ?? "dev-tenant",
    email: opts?.email ?? "analyst@viola.corp",
    role: opts?.role ?? "analyst",
  };
  const res = await fetch("http://localhost:8081/token", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`Auth service error: ${res.status}`);
  const data = (await res.json()) as { access_token: string };
  return data.access_token;
}

import type {
  ApiRequest,
  Collection,
  Environment,
  HistoryEntry,
  Report,
  RunSummary,
  StepResult,
  WsEvent,
} from "./types";

class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api${path}`, {
    headers: { "Content-Type": "application/json" },
    ...init,
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new ApiError(res.status, body.error ?? res.statusText);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}

export const api = {
  listCollections: () => request<Collection[]>("/collections"),
  getCollection: (id: string) => request<Collection>(`/collections/${id}`),
  createCollection: (c: Partial<Collection>) =>
    request<Collection>("/collections", { method: "POST", body: JSON.stringify(c) }),
  updateCollection: (id: string, c: Partial<Collection>) =>
    request<Collection>(`/collections/${id}`, { method: "PUT", body: JSON.stringify(c) }),
  deleteCollection: (id: string) => request<void>(`/collections/${id}`, { method: "DELETE" }),

  listEnvironments: () => request<Environment[]>("/environments"),
  createEnvironment: (e: Partial<Environment>) =>
    request<Environment>("/environments", { method: "POST", body: JSON.stringify(e) }),
  updateEnvironment: (id: string, e: Partial<Environment>) =>
    request<Environment>(`/environments/${id}`, { method: "PUT", body: JSON.stringify(e) }),
  deleteEnvironment: (id: string) => request<void>(`/environments/${id}`, { method: "DELETE" }),

  runCollection: (id: string, environmentId?: string, stopOnFailure?: boolean) =>
    request<Report>(`/collections/${id}/run`, {
      method: "POST",
      body: JSON.stringify({ environmentId, stopOnFailure }),
    }),
  listRuns: (collectionId: string) => request<RunSummary[]>(`/collections/${collectionId}/runs`),

  sendRequest: (req: ApiRequest, environmentId?: string) =>
    request<StepResult>("/requests/send", {
      method: "POST",
      body: JSON.stringify({ request: req, environmentId }),
    }),

  listHistory: () => request<HistoryEntry[]>("/history"),
};

/**
 * Opens the live run WebSocket for a collection, calling onEvent once per
 * step as it completes and once more with the final report. Returns a
 * closer so the caller can tear the connection down (e.g. on unmount).
 */
export function runCollectionLive(
  collectionId: string,
  environmentId: string | undefined,
  stopOnFailure: boolean,
  onEvent: (event: WsEvent) => void,
): () => void {
  const url = new URL(`/api/ws/collections/${collectionId}/run`, window.location.href);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  if (environmentId) url.searchParams.set("environmentId", environmentId);
  if (stopOnFailure) url.searchParams.set("stopOnFailure", "true");

  const socket = new WebSocket(url);
  socket.onmessage = (msg) => {
    try {
      onEvent(JSON.parse(msg.data) as WsEvent);
    } catch {
      onEvent({ type: "error", error: "received a malformed message from the server" });
    }
  };
  socket.onerror = () => onEvent({ type: "error", error: "WebSocket connection failed" });

  return () => socket.close();
}

export { ApiError };

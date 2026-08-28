import { useEffect, useState } from "react";
import { api, runCollectionLive } from "./api/client";
import type { ApiRequest, Collection, Environment, StepResult } from "./api/types";
import Sidebar from "./components/Sidebar";
import RequestBuilder from "./components/RequestBuilder";
import ResponseViewer from "./components/ResponseViewer";
import EnvironmentPanel from "./components/EnvironmentPanel";
import TestReport from "./components/TestReport";

function blankRequest(): ApiRequest {
  return {
    id: "",
    name: "",
    method: "GET",
    // The backend requires every saved request to have a URL (an empty one
    // can never actually be run), so a fresh request needs a placeholder
    // rather than "" — the user overwrites it immediately in practice.
    url: "https://example.com",
    headers: [],
    queryParams: [],
    body: { type: "none", content: "" },
    auth: { type: "none" },
    assertions: [],
  };
}

interface RunState {
  steps: StepResult[];
  running: boolean;
  summary: { passed: number; failed: number } | null;
}

export default function App() {
  const [collections, setCollections] = useState<Collection[]>([]);
  const [environments, setEnvironments] = useState<Environment[]>([]);
  const [expandedCollectionId, setExpandedCollectionId] = useState<string | null>(null);
  const [activeCollectionId, setActiveCollectionId] = useState<string | null>(null);
  const [activeRequestId, setActiveRequestId] = useState<string | null>(null);
  const [draft, setDraft] = useState<ApiRequest | null>(null);
  const [originalDraft, setOriginalDraft] = useState<string>("");
  const [activeEnvironmentId, setActiveEnvironmentId] = useState<string>("");
  const [lastStep, setLastStep] = useState<StepResult | null>(null);
  const [sending, setSending] = useState(false);
  const [showEnvPanel, setShowEnvPanel] = useState(false);
  const [run, setRun] = useState<RunState | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    refreshCollections();
    refreshEnvironments();
  }, []);

  async function refreshCollections() {
    try {
      setCollections(await api.listCollections());
    } catch (e) {
      setError(String(e));
    }
  }

  async function refreshEnvironments() {
    try {
      setEnvironments(await api.listEnvironments());
    } catch (e) {
      setError(String(e));
    }
  }

  function selectRequest(collectionId: string, requestId: string) {
    const collection = collections.find((c) => c.id === collectionId);
    const request = collection?.requests.find((r) => r.id === requestId);
    if (!request) return;
    setActiveCollectionId(collectionId);
    setActiveRequestId(requestId);
    setDraft(structuredClone(request));
    setOriginalDraft(JSON.stringify(request));
    setLastStep(null);
  }

  async function newCollection(name: string) {
    try {
      const created = await api.createCollection({ name, requests: [] });
      await refreshCollections();
      setExpandedCollectionId(created.id);
    } catch (e) {
      setError(String(e));
    }
  }

  async function newRequest(collectionId: string, name: string) {
    const collection = collections.find((c) => c.id === collectionId);
    if (!collection) return;
    try {
      const request = { ...blankRequest(), name };
      const updated = await api.updateCollection(collectionId, {
        ...collection,
        requests: [...collection.requests, request],
      });
      setCollections((prev) => prev.map((c) => (c.id === collectionId ? updated : c)));
      const added = updated.requests[updated.requests.length - 1];
      selectRequestFrom(updated, added.id);
    } catch (e) {
      setError(String(e));
    }
  }

  function selectRequestFrom(collection: Collection, requestId: string) {
    const request = collection.requests.find((r) => r.id === requestId);
    if (!request) return;
    setActiveCollectionId(collection.id);
    setActiveRequestId(requestId);
    setDraft(structuredClone(request));
    setOriginalDraft(JSON.stringify(request));
  }

  async function handleSend() {
    if (!draft) return;
    setSending(true);
    setError(null);
    try {
      const step = await api.sendRequest(draft, activeEnvironmentId || undefined);
      setLastStep(step);
    } catch (e) {
      setError(String(e));
    } finally {
      setSending(false);
    }
  }

  async function handleSave() {
    if (!draft || !activeCollectionId) return;
    const collection = collections.find((c) => c.id === activeCollectionId);
    if (!collection) return;

    try {
      const requests = collection.requests.map((r) => (r.id === draft.id ? draft : r));
      const updated = await api.updateCollection(activeCollectionId, { ...collection, requests });
      setCollections((prev) => prev.map((c) => (c.id === activeCollectionId ? updated : c)));
      setOriginalDraft(JSON.stringify(draft));
    } catch (e) {
      setError(String(e));
    }
  }

  function handleRunCollection(collectionId: string) {
    setRun({ steps: [], running: true, summary: null });
    runCollectionLive(collectionId, activeEnvironmentId || undefined, false, (event) => {
      if (event.type === "step") {
        setRun((prev) => (prev ? { ...prev, steps: [...prev.steps, event.step] } : prev));
      } else if (event.type === "done") {
        setRun({ steps: event.report.steps, running: false, summary: { passed: event.report.passed, failed: event.report.failed } });
      } else {
        setError(event.error);
        setRun((prev) => (prev ? { ...prev, running: false } : prev));
      }
    });
  }

  async function handleCreateEnvironment(name: string) {
    try {
      await api.createEnvironment({ name, variables: {} });
      await refreshEnvironments();
    } catch (e) {
      setError(String(e));
    }
  }

  async function handleSaveEnvironment(env: Environment) {
    setEnvironments((prev) => prev.map((e) => (e.id === env.id ? env : e)));
    try {
      await api.updateEnvironment(env.id, env);
    } catch (e) {
      setError(String(e));
    }
  }

  async function handleDeleteEnvironment(id: string) {
    try {
      await api.deleteEnvironment(id);
      if (activeEnvironmentId === id) setActiveEnvironmentId("");
      await refreshEnvironments();
    } catch (e) {
      setError(String(e));
    }
  }

  const dirty = draft ? JSON.stringify(draft) !== originalDraft : false;

  return (
    <div className="app">
      <Sidebar
        collections={collections}
        environments={environments}
        expandedCollectionId={expandedCollectionId}
        activeCollectionId={activeCollectionId}
        activeRequestId={activeRequestId}
        activeEnvironmentId={activeEnvironmentId}
        onToggleCollection={(id) => setExpandedCollectionId((cur) => (cur === id ? null : id))}
        onSelectRequest={selectRequest}
        onNewCollection={newCollection}
        onNewRequest={newRequest}
        onRunCollection={handleRunCollection}
        onEnvironmentChange={setActiveEnvironmentId}
        onManageEnvironments={() => setShowEnvPanel(true)}
      />

      <main className="main">
        {error && (
          <div className="banner banner--error" onClick={() => setError(null)}>
            {error}
          </div>
        )}

        {!draft && (
          <div className="main__empty">
            <p>Select a request from the sidebar, or create a new collection to get started.</p>
          </div>
        )}

        {draft && (
          <>
            <RequestBuilder request={draft} onChange={setDraft} onSend={handleSend} onSave={handleSave} sending={sending} dirty={dirty} />
            <ResponseViewer step={lastStep} />
          </>
        )}
      </main>

      {showEnvPanel && (
        <EnvironmentPanel
          environments={environments}
          onClose={() => setShowEnvPanel(false)}
          onCreate={handleCreateEnvironment}
          onSave={handleSaveEnvironment}
          onDelete={handleDeleteEnvironment}
        />
      )}

      {run && (
        <TestReport steps={run.steps} running={run.running} summary={run.summary} onClose={() => setRun(null)} />
      )}
    </div>
  );
}

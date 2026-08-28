import { useState } from "react";
import type { Collection, Environment } from "../api/types";

interface Props {
  collections: Collection[];
  environments: Environment[];
  expandedCollectionId: string | null;
  activeCollectionId: string | null;
  activeRequestId: string | null;
  activeEnvironmentId: string;
  onToggleCollection: (id: string) => void;
  onSelectRequest: (collectionId: string, requestId: string) => void;
  onNewCollection: (name: string) => void;
  onNewRequest: (collectionId: string, name: string) => void;
  onRunCollection: (collectionId: string) => void;
  onEnvironmentChange: (id: string) => void;
  onManageEnvironments: () => void;
}

export default function Sidebar({
  collections,
  environments,
  expandedCollectionId,
  activeCollectionId,
  activeRequestId,
  activeEnvironmentId,
  onToggleCollection,
  onSelectRequest,
  onNewCollection,
  onNewRequest,
  onRunCollection,
  onEnvironmentChange,
  onManageEnvironments,
}: Props) {
  const [creatingCollection, setCreatingCollection] = useState(false);
  const [newCollectionName, setNewCollectionName] = useState("");
  const [addingRequestTo, setAddingRequestTo] = useState<string | null>(null);
  const [newRequestName, setNewRequestName] = useState("");

  function submitNewCollection() {
    const name = newCollectionName.trim();
    if (!name) return;
    onNewCollection(name);
    setNewCollectionName("");
    setCreatingCollection(false);
  }

  function submitNewRequest(collectionId: string) {
    const name = newRequestName.trim() || "New Request";
    onNewRequest(collectionId, name);
    setNewRequestName("");
    setAddingRequestTo(null);
  }

  return (
    <aside className="sidebar">
      <div className="sidebar__brand">ByteForge</div>

      <div className="sidebar__environment">
        <select value={activeEnvironmentId} onChange={(e) => onEnvironmentChange(e.target.value)}>
          <option value="">No environment</option>
          {environments.map((env) => (
            <option key={env.id} value={env.id}>
              {env.name}
            </option>
          ))}
        </select>
        <button type="button" className="btn btn--ghost" onClick={onManageEnvironments}>
          Manage
        </button>
      </div>

      <div className="sidebar__section-header">
        <span>Collections</span>
        <button type="button" className="btn btn--ghost" onClick={() => setCreatingCollection((v) => !v)}>
          + New
        </button>
      </div>

      {creatingCollection && (
        <div className="sidebar__new-form">
          <input
            autoFocus
            placeholder="Collection name"
            value={newCollectionName}
            onChange={(e) => setNewCollectionName(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && submitNewCollection()}
          />
          <button type="button" className="btn" onClick={submitNewCollection}>
            Create
          </button>
        </div>
      )}

      <div className="sidebar__collections">
        {collections.length === 0 && <p className="sidebar__empty">No collections yet.</p>}
        {collections.map((c) => (
          <div key={c.id} className="collection-item">
            <div
              className={`collection-item__header ${activeCollectionId === c.id ? "collection-item__header--active" : ""}`}
              onClick={() => onToggleCollection(c.id)}
            >
              <span className="collection-item__caret">{expandedCollectionId === c.id ? "▾" : "▸"}</span>
              <span className="collection-item__name">{c.name}</span>
              <button
                type="button"
                className="btn btn--icon"
                title="Run collection"
                onClick={(e) => {
                  e.stopPropagation();
                  onRunCollection(c.id);
                }}
              >
                ▶
              </button>
            </div>

            {expandedCollectionId === c.id && (
              <div className="collection-item__requests">
                {c.requests.map((r) => (
                  <div
                    key={r.id}
                    className={`request-item ${activeRequestId === r.id ? "request-item--active" : ""}`}
                    onClick={() => onSelectRequest(c.id, r.id)}
                  >
                    <span className={`request-item__method method--${r.method.toLowerCase()}`}>{r.method}</span>
                    <span className="request-item__name">{r.name || "Untitled"}</span>
                  </div>
                ))}

                {addingRequestTo === c.id ? (
                  <div className="sidebar__new-form">
                    <input
                      autoFocus
                      placeholder="Request name"
                      value={newRequestName}
                      onChange={(e) => setNewRequestName(e.target.value)}
                      onKeyDown={(e) => e.key === "Enter" && submitNewRequest(c.id)}
                    />
                    <button type="button" className="btn" onClick={() => submitNewRequest(c.id)}>
                      Add
                    </button>
                  </div>
                ) : (
                  <button
                    type="button"
                    className="btn btn--ghost request-item__add"
                    onClick={() => setAddingRequestTo(c.id)}
                  >
                    + Add request
                  </button>
                )}
              </div>
            )}
          </div>
        ))}
      </div>
    </aside>
  );
}

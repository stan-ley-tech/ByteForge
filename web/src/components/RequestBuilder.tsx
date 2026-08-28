import { useState } from "react";
import type { ApiRequest, AuthType, BodyType, HttpMethod } from "../api/types";
import KeyValueEditor from "./KeyValueEditor";

const METHODS: HttpMethod[] = ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"];
type Tab = "params" | "headers" | "body" | "auth" | "assertions";

interface Props {
  request: ApiRequest;
  onChange: (req: ApiRequest) => void;
  onSend: () => void;
  onSave: () => void;
  sending: boolean;
  dirty: boolean;
}

export default function RequestBuilder({ request, onChange, onSend, onSave, sending, dirty }: Props) {
  const [tab, setTab] = useState<Tab>("params");

  function patch(fields: Partial<ApiRequest>) {
    onChange({ ...request, ...fields });
  }

  return (
    <div className="request-builder">
      <div className="request-builder__name">
        <input
          className="request-builder__name-input"
          value={request.name}
          placeholder="Untitled request"
          onChange={(e) => patch({ name: e.target.value })}
        />
        <button type="button" className="btn" onClick={onSave} disabled={!dirty}>
          Save
        </button>
      </div>

      <div className="request-builder__url-row">
        <select value={request.method} onChange={(e) => patch({ method: e.target.value as HttpMethod })}>
          {METHODS.map((m) => (
            <option key={m} value={m}>
              {m}
            </option>
          ))}
        </select>
        <input
          className="request-builder__url"
          value={request.url}
          placeholder="https://api.example.com/users/{{user_id}}"
          onChange={(e) => patch({ url: e.target.value })}
        />
        <button type="button" className="btn btn--primary" onClick={onSend} disabled={sending || !request.url}>
          {sending ? "Sending…" : "Send"}
        </button>
      </div>

      <div className="tabs">
        {(["params", "headers", "body", "auth", "assertions"] as Tab[]).map((t) => (
          <button
            key={t}
            type="button"
            className={`tabs__item ${tab === t ? "tabs__item--active" : ""}`}
            onClick={() => setTab(t)}
          >
            {t === "assertions" ? "Tests" : t[0].toUpperCase() + t.slice(1)}
          </button>
        ))}
      </div>

      <div className="tab-panel">
        {tab === "params" && (
          <KeyValueEditor
            label="Query Params"
            rows={request.queryParams ?? []}
            onChange={(rows) => patch({ queryParams: rows })}
          />
        )}

        {tab === "headers" && (
          <KeyValueEditor label="Headers" rows={request.headers ?? []} onChange={(rows) => patch({ headers: rows })} />
        )}

        {tab === "body" && (
          <div className="body-editor">
            <select
              value={request.body?.type ?? "none"}
              onChange={(e) => patch({ body: { type: e.target.value as BodyType, content: request.body?.content ?? "" } })}
            >
              <option value="none">No Body</option>
              <option value="json">JSON</option>
              <option value="xml">XML</option>
              <option value="form">Form URL-Encoded</option>
              <option value="raw">Raw</option>
            </select>
            {request.body && request.body.type !== "none" && (
              <textarea
                className="body-editor__content"
                value={request.body.content ?? ""}
                placeholder={request.body.type === "json" ? '{\n  "key": "value"\n}' : ""}
                onChange={(e) => patch({ body: { type: request.body!.type, content: e.target.value } })}
              />
            )}
          </div>
        )}

        {tab === "auth" && (
          <div className="auth-editor">
            <select
              value={request.auth?.type ?? "none"}
              onChange={(e) => patch({ auth: { type: e.target.value as AuthType } })}
            >
              <option value="none">No Auth</option>
              <option value="bearer">Bearer Token</option>
              <option value="basic">Basic Auth</option>
              <option value="apikey">API Key</option>
            </select>

            {request.auth?.type === "bearer" && (
              <input
                placeholder="{{ACCESS_TOKEN}}"
                value={request.auth.token ?? ""}
                onChange={(e) => patch({ auth: { ...request.auth!, token: e.target.value } })}
              />
            )}

            {request.auth?.type === "basic" && (
              <>
                <input
                  placeholder="Username"
                  value={request.auth.username ?? ""}
                  onChange={(e) => patch({ auth: { ...request.auth!, username: e.target.value } })}
                />
                <input
                  placeholder="Password"
                  type="password"
                  value={request.auth.password ?? ""}
                  onChange={(e) => patch({ auth: { ...request.auth!, password: e.target.value } })}
                />
              </>
            )}

            {request.auth?.type === "apikey" && (
              <>
                <input
                  placeholder="Key name (e.g. X-API-Key)"
                  value={request.auth.keyName ?? ""}
                  onChange={(e) => patch({ auth: { ...request.auth!, keyName: e.target.value } })}
                />
                <input
                  placeholder="{{API_KEY}}"
                  value={request.auth.keyValue ?? ""}
                  onChange={(e) => patch({ auth: { ...request.auth!, keyValue: e.target.value } })}
                />
                <select
                  value={request.auth.in ?? "header"}
                  onChange={(e) => patch({ auth: { ...request.auth!, in: e.target.value as "header" | "query" } })}
                >
                  <option value="header">Header</option>
                  <option value="query">Query Param</option>
                </select>
              </>
            )}
          </div>
        )}

        {tab === "assertions" && (
          <div className="assertions-editor">
            <p className="assertions-editor__hint">One assertion per line, e.g. <code>status == 200</code></p>
            <textarea
              className="assertions-editor__content"
              value={(request.assertions ?? []).join("\n")}
              placeholder={"status == 200\nresponse.body.id exists\nresponse.time < 500ms"}
              onChange={(e) => patch({ assertions: e.target.value.split("\n") })}
            />
          </div>
        )}
      </div>
    </div>
  );
}

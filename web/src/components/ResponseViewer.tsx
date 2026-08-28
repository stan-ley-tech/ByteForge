import { useMemo, useState } from "react";
import type { StepResult } from "../api/types";

type Tab = "body" | "headers" | "tests";

function formatBody(raw: string | undefined): { text: string; isJson: boolean } {
  if (!raw) return { text: "", isJson: false };
  try {
    return { text: JSON.stringify(JSON.parse(raw), null, 2), isJson: true };
  } catch {
    return { text: raw, isJson: false };
  }
}

function statusClass(status?: number): string {
  if (!status) return "status--error";
  if (status < 300) return "status--ok";
  if (status < 400) return "status--redirect";
  return "status--error";
}

interface Props {
  step: StepResult | null;
}

/** Shows the outcome of the last request sent: status, timing, headers, body, and assertion results. */
export default function ResponseViewer({ step }: Props) {
  const [tab, setTab] = useState<Tab>("body");
  const body = useMemo(() => formatBody(step?.body), [step?.body]);

  if (!step) {
    return (
      <div className="response-viewer response-viewer--empty">
        <p>Send a request to see its response here.</p>
      </div>
    );
  }

  if (step.error) {
    return (
      <div className="response-viewer">
        <div className="response-viewer__summary">
          <span className="status status--error">Error</span>
        </div>
        <pre className="response-viewer__error">{step.error}</pre>
      </div>
    );
  }

  const headerEntries = Object.entries(step.headers ?? {});

  return (
    <div className="response-viewer">
      <div className="response-viewer__summary">
        <span className={`status ${statusClass(step.status)}`}>{step.status}</span>
        <span className="response-viewer__timing">{step.durationMs}ms</span>
      </div>

      <div className="tabs">
        {(["body", "headers", "tests"] as Tab[]).map((t) => (
          <button
            key={t}
            type="button"
            className={`tabs__item ${tab === t ? "tabs__item--active" : ""}`}
            onClick={() => setTab(t)}
          >
            {t === "headers" && headerEntries.length > 0
              ? `Headers (${headerEntries.length})`
              : t === "tests"
                ? `Tests (${step.assertions?.length ?? 0})`
                : t[0].toUpperCase() + t.slice(1)}
          </button>
        ))}
      </div>

      <div className="tab-panel">
        {tab === "body" && (
          <pre className={`response-viewer__body ${body.isJson ? "response-viewer__body--json" : ""}`}>
            {body.text || "(empty body)"}
          </pre>
        )}

        {tab === "headers" && (
          <table className="header-table">
            <tbody>
              {headerEntries.length === 0 && (
                <tr>
                  <td className="header-table__empty" colSpan={2}>
                    No response headers.
                  </td>
                </tr>
              )}
              {headerEntries.map(([name, values]) => (
                <tr key={name}>
                  <td className="header-table__name">{name}</td>
                  <td>{values.join(", ")}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        {tab === "tests" && (
          <ul className="assertion-list">
            {(step.assertions ?? []).length === 0 && (
              <li className="assertion-list__empty">No assertions on this request.</li>
            )}
            {(step.assertions ?? []).map((a, i) => (
              <li key={i} className={a.passed ? "assertion--pass" : "assertion--fail"}>
                <span className="assertion__mark">{a.passed ? "✓" : "✗"}</span>
                <span>{a.message}</span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

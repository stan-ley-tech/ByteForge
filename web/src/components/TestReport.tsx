import type { StepResult } from "../api/types";

interface Props {
  steps: StepResult[];
  running: boolean;
  summary: { passed: number; failed: number } | null;
  onClose: () => void;
}

/** Live/finished collection run output: one card per request, assertions listed underneath. */
export default function TestReport({ steps, running, summary, onClose }: Props) {
  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal modal--wide" onClick={(e) => e.stopPropagation()}>
        <div className="modal__header">
          <h2>Test Run{running ? " (running…)" : ""}</h2>
          <button type="button" className="btn btn--icon" onClick={onClose}>
            ×
          </button>
        </div>

        <div className="test-report">
          {steps.length === 0 && !running && <p className="sidebar__empty">No steps yet.</p>}

          {steps.map((step, i) => (
            <div key={i} className={`test-report__step ${step.passed ? "test-report__step--pass" : "test-report__step--fail"}`}>
              <div className="test-report__step-header">
                <span className={`request-item__method method--${step.method.toLowerCase()}`}>{step.method}</span>
                <span className="test-report__request-name">{step.request}</span>
                {step.status !== undefined && <span className="test-report__status">{step.status}</span>}
                <span className="test-report__duration">{step.durationMs}ms</span>
              </div>

              {step.error && <p className="test-report__error">{step.error}</p>}

              {(step.assertions ?? []).map((a, j) => (
                <div key={j} className={a.passed ? "assertion--pass" : "assertion--fail"}>
                  <span className="assertion__mark">{a.passed ? "✓" : "✗"}</span> {a.message}
                </div>
              ))}
            </div>
          ))}

          {running && <div className="test-report__spinner">Running…</div>}

          {summary && (
            <div className={`test-report__summary ${summary.failed === 0 ? "test-report__summary--pass" : "test-report__summary--fail"}`}>
              {summary.passed}/{summary.passed + summary.failed} PASSED
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

import type { KeyValue } from "../api/types";

interface Props {
  label: string;
  rows: KeyValue[];
  onChange: (rows: KeyValue[]) => void;
}

/** A small editable table for headers and query parameters: enable checkbox, key, value, remove. */
export default function KeyValueEditor({ label, rows, onChange }: Props) {
  function update(index: number, patch: Partial<KeyValue>) {
    const next = rows.slice();
    next[index] = { ...next[index], ...patch };
    onChange(next);
  }

  function remove(index: number) {
    onChange(rows.filter((_, i) => i !== index));
  }

  function add() {
    onChange([...rows, { key: "", value: "", enabled: true }]);
  }

  return (
    <div className="kv-editor">
      <div className="kv-editor__header">
        <span>{label}</span>
        <button type="button" className="btn btn--ghost" onClick={add}>
          + Add
        </button>
      </div>
      {rows.length === 0 && <p className="kv-editor__empty">No {label.toLowerCase()} yet.</p>}
      {rows.map((row, i) => (
        <div className="kv-editor__row" key={i}>
          <input
            type="checkbox"
            checked={row.enabled}
            onChange={(e) => update(i, { enabled: e.target.checked })}
            aria-label={`enable row ${i + 1}`}
          />
          <input
            type="text"
            placeholder="Key"
            value={row.key}
            onChange={(e) => update(i, { key: e.target.value })}
          />
          <input
            type="text"
            placeholder="Value"
            value={row.value}
            onChange={(e) => update(i, { value: e.target.value })}
          />
          <button type="button" className="btn btn--icon" onClick={() => remove(i)} aria-label="remove row">
            ×
          </button>
        </div>
      ))}
    </div>
  );
}

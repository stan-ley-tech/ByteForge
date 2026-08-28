import { useState } from "react";
import type { Environment, Variable } from "../api/types";

interface Props {
  environments: Environment[];
  onClose: () => void;
  onCreate: (name: string) => void;
  onSave: (env: Environment) => void;
  onDelete: (id: string) => void;
}

/** Modal for creating environments and editing their variables, with a per-variable secret toggle. */
export default function EnvironmentPanel({ environments, onClose, onCreate, onSave, onDelete }: Props) {
  const [newName, setNewName] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(environments[0]?.id ?? null);
  const selected = environments.find((e) => e.id === selectedId) ?? null;

  function updateVariable(name: string, patch: Partial<Variable>) {
    if (!selected) return;
    const variables = { ...selected.variables, [name]: { ...selected.variables[name], ...patch } };
    onSave({ ...selected, variables });
  }

  function renameVariable(oldName: string, newKey: string) {
    if (!selected || !newKey || oldName === newKey) return;
    const variables = { ...selected.variables };
    variables[newKey] = variables[oldName];
    delete variables[oldName];
    onSave({ ...selected, variables });
  }

  function addVariable() {
    if (!selected) return;
    let name = "NEW_VARIABLE";
    let n = 1;
    while (selected.variables[name]) name = `NEW_VARIABLE_${n++}`;
    onSave({ ...selected, variables: { ...selected.variables, [name]: { value: "", secret: false } } });
  }

  function removeVariable(name: string) {
    if (!selected) return;
    const variables = { ...selected.variables };
    delete variables[name];
    onSave({ ...selected, variables });
  }

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal__header">
          <h2>Environments</h2>
          <button type="button" className="btn btn--icon" onClick={onClose}>
            ×
          </button>
        </div>

        <div className="env-panel">
          <div className="env-panel__list">
            {environments.map((e) => (
              <div
                key={e.id}
                className={`env-panel__item ${selectedId === e.id ? "env-panel__item--active" : ""}`}
                onClick={() => setSelectedId(e.id)}
              >
                {e.name}
              </div>
            ))}
            <div className="env-panel__new">
              <input
                placeholder="Environment name"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && newName.trim()) {
                    onCreate(newName.trim());
                    setNewName("");
                  }
                }}
              />
              <button
                type="button"
                className="btn"
                onClick={() => {
                  if (newName.trim()) {
                    onCreate(newName.trim());
                    setNewName("");
                  }
                }}
              >
                Add
              </button>
            </div>
          </div>

          <div className="env-panel__editor">
            {!selected && <p className="sidebar__empty">Select or create an environment.</p>}
            {selected && (
              <>
                <div className="env-panel__editor-header">
                  <h3>{selected.name}</h3>
                  <button type="button" className="btn btn--danger" onClick={() => onDelete(selected.id)}>
                    Delete
                  </button>
                </div>
                <table className="env-table">
                  <thead>
                    <tr>
                      <th>Variable</th>
                      <th>Value</th>
                      <th>Secret</th>
                      <th />
                    </tr>
                  </thead>
                  <tbody>
                    {Object.entries(selected.variables).map(([name, v]) => (
                      <tr key={name}>
                        <td>
                          <input defaultValue={name} onBlur={(e) => renameVariable(name, e.target.value)} />
                        </td>
                        <td>
                          <input
                            type={v.secret ? "password" : "text"}
                            value={v.value}
                            onChange={(e) => updateVariable(name, { value: e.target.value })}
                          />
                        </td>
                        <td className="env-table__secret">
                          <input
                            type="checkbox"
                            checked={v.secret}
                            onChange={(e) => updateVariable(name, { secret: e.target.checked })}
                          />
                        </td>
                        <td>
                          <button type="button" className="btn btn--icon" onClick={() => removeVariable(name)}>
                            ×
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                <button type="button" className="btn btn--ghost" onClick={addVariable}>
                  + Add variable
                </button>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

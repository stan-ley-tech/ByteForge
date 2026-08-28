import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import KeyValueEditor from "./KeyValueEditor";
import type { KeyValue } from "../api/types";

describe("KeyValueEditor", () => {
  it("renders an empty state when there are no rows", () => {
    render(<KeyValueEditor label="Headers" rows={[]} onChange={vi.fn()} />);
    expect(screen.getByText(/no headers yet/i)).toBeInTheDocument();
  });

  it("renders one row per entry", () => {
    const rows: KeyValue[] = [
      { key: "Content-Type", value: "application/json", enabled: true },
      { key: "X-Debug", value: "1", enabled: false },
    ];
    render(<KeyValueEditor label="Headers" rows={rows} onChange={vi.fn()} />);
    expect(screen.getByDisplayValue("Content-Type")).toBeInTheDocument();
    expect(screen.getByDisplayValue("X-Debug")).toBeInTheDocument();
  });

  it("calls onChange with an appended blank row when + Add is clicked", async () => {
    const onChange = vi.fn();
    render(<KeyValueEditor label="Query Params" rows={[]} onChange={onChange} />);

    await userEvent.click(screen.getByRole("button", { name: /add/i }));

    expect(onChange).toHaveBeenCalledWith([{ key: "", value: "", enabled: true }]);
  });

  it("calls onChange with the row removed when × is clicked", async () => {
    const onChange = vi.fn();
    const rows: KeyValue[] = [{ key: "A", value: "1", enabled: true }];
    render(<KeyValueEditor label="Headers" rows={rows} onChange={onChange} />);

    await userEvent.click(screen.getByRole("button", { name: /remove row/i }));

    expect(onChange).toHaveBeenCalledWith([]);
  });

  it("updates a row's value in place when typed into", async () => {
    const onChange = vi.fn();
    const rows: KeyValue[] = [{ key: "X", value: "", enabled: true }];
    render(<KeyValueEditor label="Headers" rows={rows} onChange={onChange} />);

    await userEvent.type(screen.getByPlaceholderText("Value"), "y");

    expect(onChange).toHaveBeenLastCalledWith([{ key: "X", value: "y", enabled: true }]);
  });
});

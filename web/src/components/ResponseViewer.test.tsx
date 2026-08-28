import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import ResponseViewer from "./ResponseViewer";
import type { StepResult } from "../api/types";

describe("ResponseViewer", () => {
  it("shows a placeholder when no step has run yet", () => {
    render(<ResponseViewer step={null} />);
    expect(screen.getByText(/send a request to see its response/i)).toBeInTheDocument();
  });

  it("shows the transport error instead of a status when the request failed", () => {
    const step: StepResult = { request: "R", method: "GET", url: "u", durationMs: 0, passed: false, error: "connection refused" };
    render(<ResponseViewer step={step} />);
    expect(screen.getByText("connection refused")).toBeInTheDocument();
  });

  it("pretty-prints a JSON body", () => {
    const step: StepResult = {
      request: "R",
      method: "GET",
      url: "u",
      status: 200,
      durationMs: 12,
      body: '{"id":1,"name":"a"}',
      passed: true,
    };
    render(<ResponseViewer step={step} />);
    expect(screen.getByText(/"id": 1/)).toBeInTheDocument();
  });

  // Regression test: assertions.Result used to serialize with capitalized
  // Go field names (Passed, Message), so a.passed was always undefined and
  // every assertion rendered as failed regardless of the real outcome. This
  // pins the UI-side contract: a passed:true assertion must render a pass mark.
  it("renders a passed assertion with a pass mark, not a fail mark", async () => {
    const step: StepResult = {
      request: "R",
      method: "GET",
      url: "u",
      status: 200,
      durationMs: 5,
      passed: true,
      assertions: [{ expression: "status == 200", passed: true, message: "200 == 200" }],
    };
    render(<ResponseViewer step={step} />);

    await userEvent.click(screen.getByRole("button", { name: /tests/i }));

    const item = screen.getByText("200 == 200").closest("li");
    expect(item).toHaveClass("assertion--pass");
    expect(item).not.toHaveClass("assertion--fail");
  });

  it("renders a failed assertion with a fail mark", async () => {
    const step: StepResult = {
      request: "R",
      method: "GET",
      url: "u",
      status: 500,
      durationMs: 5,
      passed: false,
      assertions: [{ expression: "status == 200", passed: false, message: "got 500, expected == 200" }],
    };
    render(<ResponseViewer step={step} />);

    await userEvent.click(screen.getByRole("button", { name: /tests/i }));

    const item = screen.getByText(/got 500/).closest("li");
    expect(item).toHaveClass("assertion--fail");
  });
});

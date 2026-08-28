import { test, expect, type Page } from "@playwright/test";

// Both tests share one backend/database (a single webServer instance), so
// collections created by one test are visible in the other's sidebar too.
// Scoping every interaction to the row for this test's own collection name
// keeps the tests correct regardless of what else exists in the sidebar.
function collectionRow(page: Page, name: string) {
  return page.locator(".collection-item", { hasText: name });
}

test("build a request, send it, add assertions, save, and run the collection", async ({ page, baseURL }) => {
  const collectionName = "E2E Collection";

  await page.goto("/");
  await expect(page.getByText("ByteForge")).toBeVisible();

  // Create a collection.
  await page.getByRole("button", { name: "+ New" }).click();
  await page.getByPlaceholder("Collection name").fill(collectionName);
  await page.getByRole("button", { name: "Create" }).click();

  const row = collectionRow(page, collectionName);

  // Add a request to it.
  await row.getByRole("button", { name: "+ Add request" }).click();
  await page.getByPlaceholder("Request name").fill("Health Check");
  await page.getByRole("button", { name: "Add" }).click();

  // Point it at the server's own health endpoint so the test is
  // self-contained: no external network dependency, deterministic response.
  await page.getByPlaceholder(/api\.example\.com/).fill(`${baseURL}/healthz`);

  // Send it and confirm a real response came back.
  await page.getByRole("button", { name: "Send" }).click();
  await expect(page.getByText("200", { exact: true })).toBeVisible();

  // Add assertions and save.
  await page.getByRole("button", { name: "Tests", exact: true }).click();
  await page.getByPlaceholder(/status == 200/).fill('status == 200\nresponse.body.status == "ok"');
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page.getByRole("button", { name: "Save" })).toBeDisabled();

  // Run the collection over the live WebSocket and check the report.
  await row.getByRole("button", { name: "Run collection" }).click();
  await expect(page.getByText("1/1 PASSED")).toBeVisible({ timeout: 10_000 });
});

test("a failing assertion is reported as a failure, not silently passed", async ({ page, baseURL }) => {
  const collectionName = "E2E Failing Collection";

  await page.goto("/");

  await page.getByRole("button", { name: "+ New" }).click();
  await page.getByPlaceholder("Collection name").fill(collectionName);
  await page.getByRole("button", { name: "Create" }).click();

  const row = collectionRow(page, collectionName);

  await row.getByRole("button", { name: "+ Add request" }).click();
  await page.getByPlaceholder("Request name").fill("Wrong Expectation");
  await page.getByRole("button", { name: "Add" }).click();

  await page.getByPlaceholder(/api\.example\.com/).fill(`${baseURL}/healthz`);

  await page.getByRole("button", { name: "Tests", exact: true }).click();
  await page.getByPlaceholder(/status == 200/).fill("status == 404");
  await page.getByRole("button", { name: "Save" }).click();

  await row.getByRole("button", { name: "Run collection" }).click();
  await expect(page.getByText(/0\/1 PASSED/)).toBeVisible({ timeout: 10_000 });
});

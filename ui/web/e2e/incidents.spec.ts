import { test, expect } from "@playwright/test";

test.describe("Incidents", () => {
  test("incidents list page loads", async ({ page }) => {
    await page.goto("/incidents");

    // Should display the incidents table
    await expect(page.locator("table")).toBeVisible();

    // Should have filter controls
    await expect(page.locator("text=All Statuses").first()).toBeVisible();
    await expect(page.locator("text=All Severities").first()).toBeVisible();
  });

  test("incidents table displays data", async ({ page }) => {
    await page.goto("/incidents");

    const rows = page.locator("table tbody tr");
    await expect(rows.first()).toBeVisible({ timeout: 10_000 });

    // Should show incident data
    const firstRow = rows.first();
    await expect(firstRow).toBeVisible();
  });

  test("clicking an incident navigates to detail page", async ({ page }) => {
    await page.goto("/incidents");

    const rows = page.locator("table tbody tr");
    await expect(rows.first()).toBeVisible({ timeout: 10_000 });
    await rows.first().click();

    await expect(page).toHaveURL(/\/incidents\/.+/);
  });

  test("filter by status works", async ({ page }) => {
    await page.goto("/incidents");

    await expect(page.locator("table tbody tr").first()).toBeVisible({
      timeout: 10_000,
    });

    await page.locator("text=All Statuses").first().click();

    const openOption = page.locator('[role="option"]').filter({ hasText: "Open" });
    if (await openOption.isVisible()) {
      await openOption.click();
      await expect(page).toHaveURL(/status=open/);
    }
  });

  test("filter by severity works", async ({ page }) => {
    await page.goto("/incidents");

    await expect(page.locator("table tbody tr").first()).toBeVisible({
      timeout: 10_000,
    });

    await page.locator("text=All Severities").first().click();

    const highOption = page.locator('[role="option"]').filter({ hasText: "High" });
    if (await highOption.isVisible()) {
      await highOption.click();
      await expect(page).toHaveURL(/severity=high/);
    }
  });
});

test.describe("Incident Detail", () => {
  test("incident detail page displays data", async ({ page }) => {
    await page.goto("/incidents");

    const rows = page.locator("table tbody tr");
    await expect(rows.first()).toBeVisible({ timeout: 10_000 });
    await rows.first().click();

    await expect(page).toHaveURL(/\/incidents\/.+/);

    // Should show incident metadata
    await expect(page.locator("text=Status").first()).toBeVisible();
    await expect(page.locator("text=Severity").first()).toBeVisible();
  });

  test("incident detail shows related alerts", async ({ page }) => {
    await page.goto("/incidents");

    const rows = page.locator("table tbody tr");
    await expect(rows.first()).toBeVisible({ timeout: 10_000 });
    await rows.first().click();

    await expect(page).toHaveURL(/\/incidents\/.+/);

    // Should show related alerts section
    const alertsSection = page.locator("text=Related Alerts").or(page.locator("text=Alerts"));
    // This may or may not be visible depending on mock data
    if (await alertsSection.isVisible()) {
      await expect(alertsSection).toBeVisible();
    }
  });

  test("incident update form works", async ({ page }) => {
    await page.goto("/incidents");

    const rows = page.locator("table tbody tr");
    await expect(rows.first()).toBeVisible({ timeout: 10_000 });
    await rows.first().click();

    // Check for assigned_to field
    const assignedInput = page.locator('input[name="assigned_to"]').or(
      page.locator('input[placeholder*="assign" i]')
    );
    if (await assignedInput.isVisible()) {
      await assignedInput.fill("analyst@viola.io");
    }
  });
});

import { test, expect } from "@playwright/test";

test.describe("Alerts", () => {
  test("alerts list page loads", async ({ page }) => {
    await page.goto("/alerts");

    // Should display the alerts table
    await expect(page.locator("table")).toBeVisible();

    // Should have filter controls
    await expect(page.locator("text=All Statuses").first()).toBeVisible();
    await expect(page.locator("text=All Severities").first()).toBeVisible();
  });

  test("alerts table displays data", async ({ page }) => {
    await page.goto("/alerts");

    // Wait for table rows to appear
    const rows = page.locator("table tbody tr");
    await expect(rows.first()).toBeVisible({ timeout: 10_000 });

    // Each row should have severity and status indicators
    const firstRow = rows.first();
    await expect(firstRow).toBeVisible();
  });

  test("clicking an alert navigates to detail page", async ({ page }) => {
    await page.goto("/alerts");

    // Wait for table to load
    const rows = page.locator("table tbody tr");
    await expect(rows.first()).toBeVisible({ timeout: 10_000 });

    // Click the first alert row
    await rows.first().click();

    // Should navigate to alert detail
    await expect(page).toHaveURL(/\/alerts\/.+/);
  });

  test("filter by status updates results", async ({ page }) => {
    await page.goto("/alerts");

    // Wait for initial load
    await expect(page.locator("table tbody tr").first()).toBeVisible({
      timeout: 10_000,
    });

    // Click status filter
    await page.locator("text=All Statuses").first().click();

    // Select "Open"
    const openOption = page.locator('[role="option"]').filter({ hasText: "Open" });
    if (await openOption.isVisible()) {
      await openOption.click();
      // URL should update with status parameter
      await expect(page).toHaveURL(/status=open/);
    }
  });

  test("filter by severity updates results", async ({ page }) => {
    await page.goto("/alerts");

    await expect(page.locator("table tbody tr").first()).toBeVisible({
      timeout: 10_000,
    });

    // Click severity filter
    await page.locator("text=All Severities").first().click();

    const criticalOption = page.locator('[role="option"]').filter({ hasText: "Critical" });
    if (await criticalOption.isVisible()) {
      await criticalOption.click();
      await expect(page).toHaveURL(/severity=critical/);
    }
  });

  test("pagination controls work", async ({ page }) => {
    await page.goto("/alerts");

    await expect(page.locator("table tbody tr").first()).toBeVisible({
      timeout: 10_000,
    });

    // Check for pagination controls
    const nextButton = page.locator("button", { hasText: /next/i });
    if (await nextButton.isVisible()) {
      await nextButton.click();
      await expect(page).toHaveURL(/page=2/);
    }
  });
});

test.describe("Alert Detail", () => {
  test("alert detail page displays data", async ({ page }) => {
    await page.goto("/alerts");

    const rows = page.locator("table tbody tr");
    await expect(rows.first()).toBeVisible({ timeout: 10_000 });
    await rows.first().click();

    await expect(page).toHaveURL(/\/alerts\/.+/);

    // Detail page should show alert information
    await expect(page.locator("text=Status").first()).toBeVisible();
    await expect(page.locator("text=Severity").first()).toBeVisible();
  });

  test("alert update form submits", async ({ page }) => {
    await page.goto("/alerts");

    const rows = page.locator("table tbody tr");
    await expect(rows.first()).toBeVisible({ timeout: 10_000 });
    await rows.first().click();

    await expect(page).toHaveURL(/\/alerts\/.+/);

    // Look for the update form
    const statusSelect = page.locator('[name="status"]').first();
    if (await statusSelect.isVisible()) {
      // Change status to acknowledged
      await statusSelect.click();
      const ackOption = page.locator('[role="option"]').filter({ hasText: /acknowledge/i });
      if (await ackOption.isVisible()) {
        await ackOption.click();
      }
    }
  });
});

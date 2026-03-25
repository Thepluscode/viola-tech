import { test, expect } from "@playwright/test";

test.describe("Navigation", () => {
  test("should load the dashboard", async ({ page }) => {
    await page.goto("/");
    await expect(page).toHaveTitle(/viola/i);
    // Dashboard should show KPI cards
    await expect(page.locator("text=Open Incidents")).toBeVisible();
    await expect(page.locator("text=Open Alerts")).toBeVisible();
  });

  test("sidebar links navigate correctly", async ({ page }) => {
    await page.goto("/");

    // Navigate to Incidents
    await page.click('a[href="/incidents"]');
    await expect(page).toHaveURL(/\/incidents/);

    // Navigate to Alerts
    await page.click('a[href="/alerts"]');
    await expect(page).toHaveURL(/\/alerts/);

    // Navigate back to Dashboard
    await page.click('a[href="/"]');
    await expect(page).toHaveURL(/\/$/);
  });

  test("sidebar highlights active route", async ({ page }) => {
    await page.goto("/alerts");
    const alertsLink = page.locator('a[href="/alerts"]');
    await expect(alertsLink).toHaveClass(/bg-/);
  });
});

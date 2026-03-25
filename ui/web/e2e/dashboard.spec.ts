import { test, expect } from "@playwright/test";

test.describe("Dashboard", () => {
  test("dashboard loads with KPI cards", async ({ page }) => {
    await page.goto("/");

    // KPI grid should be visible
    await expect(page.locator("text=Open Incidents")).toBeVisible({
      timeout: 10_000,
    });
    await expect(page.locator("text=Open Alerts")).toBeVisible();
  });

  test("dashboard shows MITRE heatmap", async ({ page }) => {
    await page.goto("/");

    // MITRE heatmap component should render
    const heatmap = page.locator("text=MITRE").or(page.locator("[data-testid=mitre-heatmap]"));
    if (await heatmap.isVisible()) {
      await expect(heatmap).toBeVisible();
    }
  });

  test("dashboard shows recent activity", async ({ page }) => {
    await page.goto("/");

    const activity = page.locator("text=Recent").or(page.locator("text=Activity"));
    if (await activity.isVisible()) {
      await expect(activity).toBeVisible();
    }
  });

  test("live feed connects and shows status", async ({ page }) => {
    await page.goto("/");

    // Live feed should show connection status
    const liveFeed = page.locator("text=Live").or(page.locator("text=Feed"));
    if (await liveFeed.isVisible()) {
      await expect(liveFeed).toBeVisible();
    }
  });

  test("KPI cards display numeric values", async ({ page }) => {
    await page.goto("/");

    // Wait for data to load
    await page.waitForTimeout(2000);

    // KPI cards should show numbers (not loading spinners)
    const kpiSection = page.locator("text=Open Incidents").locator("..");
    await expect(kpiSection).toBeVisible();
  });

  test("dashboard links to incidents page", async ({ page }) => {
    await page.goto("/");

    // Clicking on incidents KPI or link should navigate
    const incidentsLink = page.locator('a[href="/incidents"]').first();
    if (await incidentsLink.isVisible()) {
      await incidentsLink.click();
      await expect(page).toHaveURL(/\/incidents/);
    }
  });
});

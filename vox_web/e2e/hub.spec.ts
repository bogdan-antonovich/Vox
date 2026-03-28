import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers/auth";

// ─── Create hub ───────────────────────────────────────────────────────────────

test("create hub → hub card appears on dashboard", async ({ page }) => {
  await loginAs(page);

  const before = await page
    .locator('[data-testid="hub-card"]')
    .count()
    .catch(() => page.locator("text=HUB").count());

  // Click "+ New Hub"
  await page.getByRole("button", { name: /new hub/i }).click();

  // A new hub card should appear
  await expect(page.locator("text=HUB").last()).toBeVisible({ timeout: 8_000 });

  const after = await page.locator("text=HUB").count();
  expect(after).toBeGreaterThan(before);
});

// ─── Go live (navigate to broadcast page) ────────────────────────────────────

test("go live → navigates to broadcast page with correct hub ID", async ({
  page,
}) => {
  await loginAs(page);

  // Create a fresh hub
  await page.getByRole("button", { name: /new hub/i }).click();
  await expect(page.locator("text=HUB").last()).toBeVisible({ timeout: 8_000 });

  // Click "Go Live →"
  await page
    .getByRole("button", { name: /go live/i })
    .last()
    .click();

  // Wait for navigation to complete
  await page.waitForURL("**/host/**", { timeout: 8_000 });

  // Extract hub ID from URL
  const url = page.url();
  const hubId = url.split("/host/")[1];
  expect(hubId.length).toBeGreaterThan(0);

  // Broadcast page assertions
  await expect(page).toHaveURL(new RegExp(`/host/${hubId}`));
  await expect(
    page.locator("nav").getByText(hubId, { exact: false }),
  ).toBeVisible();
  await expect(page.getByText(/ready to broadcast/i)).toBeVisible();

  // Clean up — go back and delete
  await page.getByRole("button", { name: /← back/i }).click();

  // Register dialog handler BEFORE clicking delete
  page.on("dialog", (d) => d.accept());
  await page.getByTitle("Delete hub").last().click();
});

// ─── Copy hub ID ──────────────────────────────────────────────────────────────

test("copy hub ID → button shows ✓ confirmation", async ({ page }) => {
  await loginAs(page);

  await page.getByRole("button", { name: /new hub/i }).click();
  await expect(page.locator("text=HUB").last()).toBeVisible({ timeout: 8_000 });

  await page
    .getByRole("button", { name: /copy id/i })
    .last()
    .click();

  // Button text changes to ✓ Copied
  await expect(
    page.getByRole("button", { name: /✓ copied/i }).last(),
  ).toBeVisible();
});

// ─── Delete hub ───────────────────────────────────────────────────────────────
test("delete hub → card disappears from dashboard", async ({ page }) => {
  await loginAs(page);

  await page.getByRole("button", { name: /new hub/i }).click();
  await expect(page.getByText("HUB", { exact: true }).first()).toBeVisible({
    timeout: 8_000,
  });

  const countBefore = await page.getByText("HUB", { exact: true }).count();

  page.on("dialog", (d) => d.accept());
  await page.getByTitle("Delete hub").last().click();

  await expect(page.getByText("HUB", { exact: true })).toHaveCount(
    countBefore - 1,
    { timeout: 8_000 },
  );
});

// ─── Protected route ──────────────────────────────────────────────────────────

test("unauthenticated user cannot access /host", async ({ page }) => {
  await page.goto("/vox/host");
  await page.waitForURL("**/vox", { timeout: 10_000 });
  await expect(page).toHaveURL("https://bogdanantonovich.com/vox");
});

test("unauthenticated user cannot access broadcast page directly", async ({
  page,
}) => {
  await page.goto("/vox/host/00000000-0000-0000-0000-000000000000");
  await page.waitForURL("**/vox", { timeout: 10_000 });
  await expect(page).toHaveURL("https://bogdanantonovich.com/vox");
});

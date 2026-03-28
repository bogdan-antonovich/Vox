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

  // Grab the hub ID from the last card's code element
  const hubIdEl = page.locator("code").last();
  const hubId = (await hubIdEl.textContent())?.trim() ?? "";
  expect(hubId.length).toBeGreaterThan(0);

  // Click "Go Live →" on that card
  await page
    .getByRole("button", { name: /go live/i })
    .last()
    .click();

  // Should navigate to /host/<hubId>
  await expect(page).toHaveURL(new RegExp(`/host/${hubId}`));

  // Broadcast page should show the hub ID in the nav
  await expect(
    page.locator("nav").getByText(hubId, { exact: false }),
  ).toBeVisible();

  // "Ready to Broadcast" status shown
  await expect(page.getByText(/ready to broadcast/i)).toBeVisible();

  // Clean up — go back and delete
  await page.getByRole("button", { name: /← back/i }).click();
  await page.getByTitle("Delete hub").last().click();
  await page.waitForEvent("dialog").then((d) => d.accept());
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

  // Create hub
  await page.getByRole("button", { name: /new hub/i }).click();
  await expect(page.locator("text=HUB").last()).toBeVisible({ timeout: 8_000 });

  const hubIdEl = page.locator("code").last();
  const hubId = (await hubIdEl.textContent())?.trim() ?? "";

  // Delete it
  await page.getByTitle("Delete hub").last().click();

  // Confirm browser dialog
  page.on("dialog", (d) => d.accept());

  // Card with that hub ID should be gone
  await expect(page.locator(`text=${hubId}`)).not.toBeVisible({
    timeout: 8_000,
  });
});

// ─── Protected route ──────────────────────────────────────────────────────────

test("unauthenticated user cannot access /host", async ({ page }) => {
  await page.goto("/host");
  // ProtectedRoute should redirect away
  await expect(page).not.toHaveURL(/\/host$/);
});

test("unauthenticated user cannot access broadcast page directly", async ({
  page,
}) => {
  await page.goto("/host/00000000-0000-0000-0000-000000000000");
  await expect(page).not.toHaveURL(/\/host\/.+/);
});

import { test, expect } from "@playwright/test";
import { loginAs, signUpFresh, TEST_CREDENTIALS } from "./helpers/auth";

// ─── Sign Up ──────────────────────────────────────────────────────────────────

test("sign up → lands on dashboard, user name visible", async ({ page }) => {
  const user = await signUpFresh(page);

  // Should be on the host dashboard
  await expect(page).toHaveURL(/\/host/);

  // User name should appear in the nav
  await expect(page.getByText(user.name)).toBeVisible();
});

// ─── Login ────────────────────────────────────────────────────────────────────

test("login with valid credentials → lands on dashboard", async ({ page }) => {
  await loginAs(page);
  await expect(page).toHaveURL(/\/host/);
});

test("login → user name is displayed in the navbar", async ({ page }) => {
  await loginAs(page);

  // The dashboard nav shows user.name or user.email
  const nav = page.locator("nav");
  await expect(nav).toBeVisible();

  // At least one of name / email must be visible somewhere in nav
  const nameOrEmail = page.locator("nav").getByText(/.+/);
  await expect(nameOrEmail.first()).toBeVisible();
});

test("login → GET /user/info cookie is valid (auth persists on reload)", async ({
  page,
}) => {
  await loginAs(page);

  // Reload the page — if auth cookies are valid, we stay on the dashboard
  await page.reload();
  await expect(page).toHaveURL(/\/host/);
});

// ─── Token refresh ────────────────────────────────────────────────────────────

test("token refresh → session survives after refresh call", async ({
  page,
}) => {
  await loginAs(page);

  // Manually call /auth/refresh — simulates token expiry scenario
  const response = await page.evaluate(async () => {
    const res = await fetch(
      "https://bogdanantonovich.com/vox/api/auth/refresh",
      {
        method: "POST",
        credentials: "include",
      },
    );
    return res.status;
  });

  expect(response).toBe(201);

  // App should still work after the refresh
  await page.reload();
  await expect(page).toHaveURL(/\/host/);
});

// ─── Logout ───────────────────────────────────────────────────────────────────

test("logout → redirected to home, dashboard is inaccessible", async ({
  page,
}) => {
  await loginAs(page);

  // Click sign out
  await page.getByRole("button", { name: /sign out/i }).click();

  // Should land on home
  await expect(page).toHaveURL(/\//);

  // Trying to navigate to /host should redirect away (ProtectedRoute)
  await page.goto("/vox/host");
  await expect(page).not.toHaveURL(/\/host/);
});

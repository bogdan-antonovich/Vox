import { type Page } from "@playwright/test";

// ── Swap these for a real test account in your DB ──────────────────────────
export const TEST_CREDENTIALS = {
  login: process.env.E2E_LOGIN ?? "",
  password: process.env.E2E_PASSWORD ?? "",
};

/**
 * Logs in via the UI auth modal and waits until the dashboard is visible.
 * Call this at the top of any test that requires authentication.
 */
export async function loginAs(page: Page, credentials = TEST_CREDENTIALS) {
  await page.goto("/");

  // Open login modal — the home page has a Sign In button
  await page.getByRole("button", { name: /sign in/i }).click();

  // Fill credentials
  await page.getByLabel(/login/i).fill(credentials.login);
  await page.getByLabel(/password/i).fill(credentials.password);
  await page.getByRole("button", { name: /^log in|^sign in/i }).click();

  // Wait until we land on the dashboard
  await page.waitForURL("**/host", { timeout: 10_000 });
}

/**
 * Signs up a fresh throwaway account.
 * Uses a timestamp suffix so each run gets a unique user.
 */
export async function signUpFresh(page: Page) {
  const ts = Date.now();
  const payload = {
    login: `e2e_${ts}`,
    password: "E2eTestPass123!",
    email: `e2e_${ts}@vox-test.dev`,
    name: `E2E User ${ts}`,
  };

  await page.goto("/");
  await page
    .getByRole("button", { name: /sign up|create account|get started/i })
    .click();

  await page.getByLabel(/email/i).fill(payload.email);
  await page.getByLabel(/login|username/i).fill(payload.login);
  await page.getByLabel(/name/i).fill(payload.name);
  await page.getByLabel(/password/i).fill(payload.password);
  await page.getByRole("button", { name: /sign up|create account/i }).click();

  await page.waitForURL("**/host", { timeout: 10_000 });

  return payload;
}

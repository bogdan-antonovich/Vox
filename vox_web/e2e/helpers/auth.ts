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
  await page.goto("/vox");

  // "Sign in" button in the nav opens the AuthModal
  await page.getByRole("button", { name: "Sign in" }).click();

  // Modal is now open — fill the form
  // Field is labeled "Username" in the modal
  await page.getByLabel("Username").fill(credentials.login);
  await page.getByLabel("Password").fill(credentials.password);

  // Submit button inside the form says "Sign in"
  await page.getByRole("button", { name: "Sign in" }).last().click();

  // Wait until we land on the dashboard
  await page.waitForURL("**/vox/host", { timeout: 10_000 });
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

  await page.goto("/vox");

  // Open modal via nav "Sign in" button
  await page.getByRole("button", { name: "Sign in" }).click();

  // Wait for modal to be fully rendered
  await page.waitForSelector("text=Welcome back");

  // Switch to sign up — target the toggle inside the <p> at the bottom of the modal
  await page.locator("p").getByRole("button", { name: "Sign up" }).click();

  // Wait for form to switch to sign-up mode
  await page.waitForSelector("text=Create account");

  // Fill sign up fields
  await page.getByLabel("Full name").fill(payload.name);
  await page.getByLabel("Email").fill(payload.email);
  await page.getByLabel("Username").fill(payload.login);
  await page.getByLabel("Password").fill(payload.password);

  // Submit
  await page.getByRole("button", { name: "Create account" }).click();

  await page.waitForURL("**/vox/host", { timeout: 10_000 });

  return payload;
}

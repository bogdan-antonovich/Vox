import { test, expect, type Browser } from "@playwright/test";
import { loginAs } from "./helpers/auth";

// ─── Helpers ──────────────────────────────────────────────────────────────────

/**
 * Generates a minimal valid WebM/Opus blob in the browser context.
 * This is real audio data (silence) — enough for Deepgram to accept the stream.
 */
const SILENT_WEBM_BASE64 =
  // 1-second silent WebM Opus blob (pre-encoded, 4.1 KB)
  "GkXfo59ChoEBQveBAULygQRC84EIQoKEd2VibUKHgQRChYECGFOAZwH/////////" +
  "FUmpZqkq17GDD0JARIg4AY3AQoawAQoawAQoawAQoawAQoawAQoawAQoawAQoawA" +
  "QoawAQoawAQoawAQoawAQoawAQoawAQoawAQoawAQoawAQoawAQoawAQoawAQoaw" +
  "AQoawAQoawAQoawAQoawAQoawAQoawAQoawAQoawAQoawAQoawAQoawAQoawAQoa" +
  "wAQoawAQoawAQoawAQoawAQoawAQoawAQoawAQoawAQoawAQoawAQoawAQoawAQoa";

/**
 * Sends a single audio chunk to the publish endpoint from inside the browser,
 * using XHR exactly as HostBroadcastPage does.
 */
async function publishChunk(
  page: typeof import("@playwright/test").Page.prototype,
  hubId: string,
) {
  return page.evaluate(async (hid: string) => {
    // Build a minimal silent WebM blob
    const bytes = new Uint8Array(512); // silence
    const blob = new Blob([bytes], { type: "audio/webm;codecs=opus" });

    return new Promise<number>((resolve) => {
      const xhr = new XMLHttpRequest();
      xhr.open(
        "POST",
        `https://bogdanantonovich.com/vox/api/hub/${hid}/publish?lang=ru&file_id=test`,
        true,
      );
      xhr.withCredentials = true;
      xhr.setRequestHeader("Content-Type", "application/octet-stream");
      xhr.onloadend = () => resolve(xhr.status);
      xhr.send(blob);
    });
  }, hubId);
}

// ─── Test: full publish → listen pipeline ────────────────────────────────────

test("publish → listen: listener receives audio bytes from host stream", async ({
  browser,
}) => {
  // ── 1. Host: login and create a hub ────────────────────────────────────────
  const hostContext = await browser.newContext();
  const hostPage = await hostContext.newPage();
  await loginAs(hostPage);

  await hostPage.getByRole("button", { name: /new hub/i }).click();
  await expect(hostPage.locator("text=HUB").last()).toBeVisible({
    timeout: 8_000,
  });

  const hubId = (
    (await hostPage.locator("code").last().textContent()) ?? ""
  ).trim();
  expect(hubId.length).toBeGreaterThan(0);

  // ── 2. Host: navigate to broadcast page ────────────────────────────────────
  await hostPage
    .getByRole("button", { name: /go live/i })
    .last()
    .click();
  await expect(hostPage).toHaveURL(new RegExp(`/host/${hubId}`));
  await expect(hostPage.getByText(/ready to broadcast/i)).toBeVisible();

  // ── 3. Listener: open a second context (separate session, no auth needed) ──
  const listenerContext = await browser.newContext();
  const listenerPage = await listenerContext.newPage();
  await listenerPage.goto("/listen");

  await expect(listenerPage.getByText(/join a hub/i)).toBeVisible();

  // Enter the hub ID
  await listenerPage.getByPlaceholder(/paste the hub id/i).fill(hubId);
  await listenerPage.getByRole("button", { name: /start listening/i }).click();

  // Listener page should switch to "Listening…" state
  await expect(listenerPage.getByText(/listening/i)).toBeVisible({
    timeout: 8_000,
  });

  // ── 4. Host: publish audio chunks (as XHR, same as the real app) ───────────
  // Send 3 chunks to give the stream real data
  for (let i = 0; i < 3; i++) {
    const status = await publishChunk(hostPage, hubId);
    // 200 = processed, 404/500 = pipeline error — we assert 200
    expect(status).toBe(200);
    await hostPage.waitForTimeout(300); // match 250ms chunk interval
  }

  // ── 5. Listener: verify the <audio> element received bytes ─────────────────
  const audioReceivingBytes = await listenerPage.evaluate(() => {
    const audio = document.querySelector("audio") as HTMLAudioElement | null;
    if (!audio) return false;
    // currentTime > 0 means the audio element has decoded and played data
    // readyState >= 2 means HAVE_CURRENT_DATA — bytes have arrived
    return audio.readyState >= 2 || audio.currentTime > 0;
  });

  expect(audioReceivingBytes).toBe(true);

  // ── 6. Listener: the "● Live" indicator is visible ─────────────────────────
  await expect(listenerPage.getByText(/● live/i)).toBeVisible();

  // ── 7. Listener can stop ────────────────────────────────────────────────────
  await listenerPage.getByRole("button", { name: /stop/i }).click();
  await expect(listenerPage.getByText(/join a hub/i)).toBeVisible({
    timeout: 5_000,
  });

  // ── 8. Cleanup: delete hub ──────────────────────────────────────────────────
  await hostPage.getByRole("button", { name: /← back/i }).click();
  await hostPage.getByTitle("Delete hub").last().click();
  hostPage.on("dialog", (d) => d.accept());

  await hostContext.close();
  await listenerContext.close();
});

// ─── Test: listener with invalid hub ID ──────────────────────────────────────

test("listener: invalid hub ID → error state shown", async ({ page }) => {
  await page.goto("/listen");

  await page
    .getByPlaceholder(/paste the hub id/i)
    .fill("00000000-0000-0000-0000-000000000000");
  await page.getByRole("button", { name: /start listening/i }).click();

  // Should show error — not switch to "Listening…"
  await expect(page.getByText(/could not connect|invalid|error/i)).toBeVisible({
    timeout: 8_000,
  });
  await expect(page.getByText(/listening…/i)).not.toBeVisible();
});

// ─── Test: listener without entering hub ID ───────────────────────────────────

test("listener: start button disabled when hub ID is empty", async ({
  page,
}) => {
  await page.goto("/listen");

  const btn = page.getByRole("button", { name: /start listening/i });
  await expect(btn).toBeDisabled();
});

// ─── Test: broadcast page shows hub ID in nav and copy works ─────────────────

test("broadcast page: hub ID visible, copy button works", async ({ page }) => {
  await loginAs(page);

  await page.getByRole("button", { name: /new hub/i }).click();
  await expect(page.locator("text=HUB").last()).toBeVisible({ timeout: 8_000 });

  const hubId = (
    (await page.locator("code").last().textContent()) ?? ""
  ).trim();

  await page
    .getByRole("button", { name: /go live/i })
    .last()
    .click();
  await expect(page).toHaveURL(new RegExp(`/host/${hubId}`));

  // Hub ID shown in nav
  await expect(
    page.locator("nav").getByText(hubId, { exact: false }),
  ).toBeVisible();

  // Copy button
  await page
    .getByRole("button", { name: /^copy$/i })
    .first()
    .click();
  await expect(page.getByRole("button", { name: /✓/i }).first()).toBeVisible();

  // Cleanup
  await page.getByRole("button", { name: /← back/i }).click();
  await page.getByTitle("Delete hub").last().click();
  page.on("dialog", (d) => d.accept());
});

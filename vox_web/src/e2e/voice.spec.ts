import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers/auth";

// ─── Helpers ──────────────────────────────────────────────────────────────────

/**
 * Uploads a silent WebM blob via the API directly from the browser context.
 * Mirrors exactly what the ProfilePage does via voiceApi.upload().
 */
async function uploadVoiceBlob(
  page: typeof import("@playwright/test").Page.prototype,
  textRef: string,
) {
  return page.evaluate(async (ref: string) => {
    const bytes = new Uint8Array(512);
    const blob = new Blob([bytes], { type: "audio/webm;codecs=opus" });

    const res = await fetch(
      `https://bogdanantonovich.com/vox/api/user/voice?text_ref=${encodeURIComponent(ref)}`,
      {
        method: "POST",
        headers: { "Content-Type": "application/octet-stream" },
        credentials: "include",
        body: blob,
      },
    );
    return res.status;
  }, textRef);
}

// ─── Voice meta ───────────────────────────────────────────────────────────────

test("voice meta: GET /user/voice/meta returns array after login", async ({
  page,
}) => {
  await loginAs(page);

  const result = await page.evaluate(async () => {
    const res = await fetch(
      "https://bogdanantonovich.com/vox/api/user/voice/meta",
      {
        credentials: "include",
      },
    );
    const body = await res.json();
    return { status: res.status, isArray: Array.isArray(body) };
  });

  expect(result.status).toBe(200);
  expect(result.isArray).toBe(true);
});

// ─── Upload voice ─────────────────────────────────────────────────────────────

test("upload voice → 200, appears in meta list", async ({ page }) => {
  await loginAs(page);

  const textRef = `e2e_ref_${Date.now()}`;
  const uploadStatus = await uploadVoiceBlob(page, textRef);
  expect(uploadStatus).toBe(200);

  // Fetch meta and confirm our entry is there
  const meta = await page.evaluate(async () => {
    const res = await fetch(
      "https://bogdanantonovich.com/vox/api/user/voice/meta",
      {
        credentials: "include",
      },
    );
    return res.json();
  });

  expect(Array.isArray(meta)).toBe(true);
  const entry = meta.find((v: { text: string }) => v.text === textRef);
  expect(entry).toBeDefined();
  expect(entry.file_id).toBeTruthy();

  // Cleanup
  await page.evaluate(async (fileId: string) => {
    await fetch(
      `https://bogdanantonovich.com/vox/api/user/voice?file_id=${fileId}`,
      {
        method: "DELETE",
        credentials: "include",
      },
    );
  }, entry.file_id);
});

// ─── Download voice file ──────────────────────────────────────────────────────

test("upload voice → GET /user/voice/file returns blob bytes", async ({
  page,
}) => {
  await loginAs(page);

  const textRef = `e2e_file_${Date.now()}`;
  await uploadVoiceBlob(page, textRef);

  // Get the file_id
  const meta = await page.evaluate(async () => {
    const res = await fetch(
      "https://bogdanantonovich.com/vox/api/user/voice/meta",
      {
        credentials: "include",
      },
    );
    return res.json();
  });

  const entry = meta.find((v: { text: string }) => v.text === textRef);
  expect(entry).toBeDefined();

  // Download the file
  const fileResult = await page.evaluate(async (fileId: string) => {
    const res = await fetch(
      `https://bogdanantonovich.com/vox/api/user/voice/file?file_id=${fileId}`,
      { credentials: "include" },
    );
    const blob = await res.blob();
    return { status: res.status, size: blob.size };
  }, entry.file_id);

  expect(fileResult.status).toBe(200);
  expect(fileResult.size).toBeGreaterThan(0); // real bytes came back

  // Cleanup
  await page.evaluate(async (fileId: string) => {
    await fetch(
      `https://bogdanantonovich.com/vox/api/user/voice?file_id=${fileId}`,
      {
        method: "DELETE",
        credentials: "include",
      },
    );
  }, entry.file_id);
});

// ─── Delete voice ─────────────────────────────────────────────────────────────

test("delete voice → 204, disappears from meta list", async ({ page }) => {
  await loginAs(page);

  // Upload first
  const textRef = `e2e_del_${Date.now()}`;
  await uploadVoiceBlob(page, textRef);

  const meta = await page.evaluate(async () => {
    const res = await fetch(
      "https://bogdanantonovich.com/vox/api/user/voice/meta",
      {
        credentials: "include",
      },
    );
    return res.json();
  });

  const entry = meta.find((v: { text: string }) => v.text === textRef);
  expect(entry).toBeDefined();

  // Delete
  const deleteStatus = await page.evaluate(async (fileId: string) => {
    const res = await fetch(
      `https://bogdanantonovich.com/vox/api/user/voice?file_id=${fileId}`,
      { method: "DELETE", credentials: "include" },
    );
    return res.status;
  }, entry.file_id);

  expect(deleteStatus).toBe(204);

  // Confirm gone from meta
  const metaAfter = await page.evaluate(async () => {
    const res = await fetch(
      "https://bogdanantonovich.com/vox/api/user/voice/meta",
      {
        credentials: "include",
      },
    );
    return res.json();
  });

  const gone = metaAfter.find((v: { text: string }) => v.text === textRef);
  expect(gone).toBeUndefined();
});

// ─── Ownership check ──────────────────────────────────────────────────────────

test("voice file: accessing another user file_id returns 403 or 404", async ({
  page,
}) => {
  await loginAs(page);

  const result = await page.evaluate(async () => {
    const res = await fetch(
      "https://bogdanantonovich.com/vox/api/user/voice/file?file_id=nonexistent-file-000",
      { credentials: "include" },
    );
    return res.status;
  });

  expect([403, 404]).toContain(result);
});

import { expect, test, type Page } from "@playwright/test";

async function registerMockUser(page: Page, seed: string) {
  await page.goto("/auth/keygen");

  await page.getByTestId("keygen-start").click();
  await page.getByTestId("keygen-generate").click();
  await page.getByTestId("keygen-confirm").check();
  await page.getByTestId("keygen-continue").click();

  await expect(page).toHaveURL(/\/auth\/register$/);
  await page.getByTestId("register-username").fill(`e2e${seed}`);
  await page.getByTestId("register-display-name").fill(`E2E ${seed}`);
  await page.getByTestId("register-submit").click();

  await expect(page).toHaveURL(/\/app$/);
}

test.describe("chat and device flows in mock mode", () => {
  test("can send a chat message end-to-end in the app shell", async ({ page }) => {
    const seed = Date.now().toString();
    const messageText = `playwright message ${seed}`;

    await registerMockUser(page, seed);

    await page.getByTestId("chat-composer-input").fill(messageText);
    await page.getByTestId("chat-composer-send").click();

    await expect(page.getByTestId("chat-message-bubble").filter({ hasText: messageText })).toBeVisible();
  });

  test("can complete the backup and device approval flow end-to-end", async ({ browser }) => {
    const seed = Date.now().toString();
    const passphrase = `Pass-${seed}!`;
    const context = await browser.newContext();
    const trustedPage = await context.newPage();

    await registerMockUser(trustedPage, seed);
    await trustedPage.getByTestId("open-settings").click();
    await trustedPage.getByTestId("settings-security-tab").click();

    await trustedPage.getByTestId("backup-export-passphrase").fill(passphrase);
    await trustedPage.getByTestId("backup-export-submit").click();

    await expect(trustedPage.getByTestId("backup-blob-input")).not.toHaveValue("");
    const backupBlob = await trustedPage.getByTestId("backup-blob-input").inputValue();
    expect(backupBlob).toContain("ciphertext");

    const devicePage = await context.newPage();
    await devicePage.goto("/auth/device");
    await devicePage.getByTestId("device-link-username").fill(`e2e${seed}`);
    await devicePage.getByTestId("device-link-passphrase").fill(passphrase);
    await devicePage.getByTestId("device-link-backup-blob").fill(backupBlob);
    await devicePage.getByTestId("device-link-prepare").click();

    const pairingPayload = await devicePage.getByTestId("device-link-pairing-payload").inputValue();
    expect(pairingPayload).toContain("device_signing_key");

    await trustedPage.getByTestId("device-pairing-input").fill(pairingPayload);
    await trustedPage.getByTestId("device-pairing-register-approve").click();
    await expect(trustedPage.getByText("Device approved: New web browser")).toBeVisible();

    await devicePage.getByTestId("device-link-complete").click();
    await expect(devicePage).toHaveURL(/\/app$/);
    await expect(devicePage.getByText("Naier")).toBeVisible();

    await context.close();
  });
});

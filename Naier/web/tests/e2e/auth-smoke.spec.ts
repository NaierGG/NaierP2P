import { expect, test } from "@playwright/test";

test.describe("auth smoke", () => {
  test("can generate keys, register, enter app, and open settings", async ({ page }) => {
    const seed = Date.now().toString();

    await page.goto("/auth/keygen");

    await page.getByTestId("keygen-start").click();
    await page.getByTestId("keygen-generate").click();
    await expect(page.getByTestId("keygen-confirm")).toBeVisible();
    await page.getByTestId("keygen-confirm").check();
    await page.getByTestId("keygen-continue").click();

    await expect(page).toHaveURL(/\/auth\/register$/);
    await page.getByTestId("register-username").fill(`smoke${seed}`);
    await page.getByTestId("register-display-name").fill(`Smoke ${seed}`);
    await page.getByTestId("register-submit").click();

    await expect(page).toHaveURL(/\/app$/);
    await expect(page.getByText("Naier")).toBeVisible();
    await expect(page.getByText("Mock mode")).toBeVisible();

    await page.getByTestId("open-settings").click();
    await expect(page).toHaveURL(/\/app\/settings$/);
    await page.getByTestId("settings-security-tab").click();
    await expect(page.getByText("Device pairing")).toBeVisible();
  });

  test("shows the new-device onboarding page", async ({ page }) => {
    await page.goto("/auth/device");

    await expect(page.getByText("Add a new device")).toBeVisible();
    await expect(page.getByTestId("device-link-prepare")).toBeVisible();
    await expect(page.getByTestId("device-link-complete")).toBeVisible();
  });
});

import { expect, test } from "@playwright/test";

test.describe("auth live @live", () => {
  test("can register with an invite and send a message against the real backend", async ({ page, request }) => {
    try {
      const health = await request.get("http://127.0.0.1:8080/health");
      test.skip(!health.ok(), "Live backend is not running on http://127.0.0.1:8080");
    } catch {
      test.skip(true, "Live backend is not running on http://127.0.0.1:8080");
    }

    const adminToken = process.env.PLAYWRIGHT_ADMIN_TOKEN;
    test.skip(!adminToken, "PLAYWRIGHT_ADMIN_TOKEN is required for invite-only live tests");

    const inviteResponse = await request.post("http://127.0.0.1:8080/api/v1/admin/invites", {
      headers: {
        "X-Admin-Token": adminToken!,
        "X-Admin-Actor": "playwright",
      },
      data: {
        note: "playwright live registration",
        max_uses: 1,
      },
    });
    expect(inviteResponse.ok()).toBeTruthy();
    const invitePayload = (await inviteResponse.json()) as {
      invite: { code: string };
    };

    const seed = Date.now().toString();
    const username = `live${seed}`;
    const messageText = `live message ${seed}`;

    await page.goto("/auth/keygen");

    await page.getByTestId("keygen-start").click();
    await page.getByTestId("keygen-generate").click();
    await page.getByTestId("keygen-confirm").check();
    await page.getByTestId("keygen-continue").click();

    await expect(page).toHaveURL(/\/auth\/register$/);
    await page.getByTestId("register-username").fill(username);
    await page.getByTestId("register-display-name").fill(`Live ${seed}`);
    await page.getByTestId("register-invite-code").fill(invitePayload.invite.code);
    await page.getByTestId("register-submit").click();

    await expect(page).toHaveURL(/\/app$/);
    await expect(page.getByText("Naier")).toBeVisible();
    await expect(page.getByText("Mock mode")).toHaveCount(0);

    const accessToken = await page.evaluate(() => localStorage.getItem("naier.access_token"));
    expect(accessToken).toBeTruthy();

    const createChannelResponse = await request.post("http://127.0.0.1:8080/api/v1/channels", {
      headers: {
        Authorization: `Bearer ${accessToken!}`,
      },
      data: {
        type: "group",
        name: `Live Room ${seed}`,
        description: "Playwright live room",
        is_encrypted: true,
        max_members: 8,
      },
    });
    expect(createChannelResponse.ok()).toBeTruthy();

    await page.reload();
    await expect(page.getByText(`Live Room ${seed}`)).toBeVisible();
    await page.getByText(`Live Room ${seed}`).click();
    await page.getByTestId("chat-composer-input").fill(messageText);
    await page.getByTestId("chat-composer-send").click();
    await expect(page.getByTestId("chat-message-bubble").filter({ hasText: messageText })).toBeVisible();

    await page.getByTestId("open-settings").click();
    await expect(page).toHaveURL(/\/app\/settings$/);
    await page.getByTestId("settings-security-tab").click();
    await expect(page.getByText("Encrypted backup")).toBeVisible();
    await expect(page.getByText("Device pairing")).toBeVisible();
  });
});

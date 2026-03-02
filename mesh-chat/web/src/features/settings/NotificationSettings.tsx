import type { Dispatch, SetStateAction } from "react";

import type { AppSettings } from "@/features/settings/settingsStorage";
import {
  ensureDesktopNotificationPermission,
  previewNotificationSound,
} from "@/shared/lib/browserNotifications";

interface NotificationSettingsProps {
  settings: AppSettings;
  setSettings: Dispatch<SetStateAction<AppSettings>> | ((next: AppSettings | ((current: AppSettings) => AppSettings)) => void);
}

export default function NotificationSettings({
  settings,
  setSettings,
}: NotificationSettingsProps) {
  async function updateSetting<Key extends keyof AppSettings>(key: Key, value: AppSettings[Key]) {
    if (key === "desktopNotifications") {
      await ensureDesktopNotificationPermission(Boolean(value));
    }

    if (key === "soundNotifications" && value) {
      await previewNotificationSound(true);
    }

    setSettings((current) => ({
      ...current,
      [key]: value,
    }));
  }

  return (
    <section className="settings-panel">
      <p className="eyebrow">Notifications</p>
      <h2>Alerts and chat behavior</h2>
      <div className="settings-field-grid">
        <div className="settings-switch">
          <label>
            <input
              checked={settings.desktopNotifications}
              onChange={(event) => {
                void updateSetting("desktopNotifications", event.target.checked);
              }}
              type="checkbox"
            />
            <span>
              <strong>Desktop notifications</strong>
              <span className="muted">Show browser notifications for new activity.</span>
            </span>
          </label>
        </div>

        <div className="settings-switch">
          <label>
            <input
              checked={settings.soundNotifications}
              onChange={(event) => {
                void updateSetting("soundNotifications", event.target.checked);
              }}
              type="checkbox"
            />
            <span>
              <strong>Sound alerts</strong>
              <span className="muted">Play a sound when new messages arrive.</span>
            </span>
          </label>
        </div>

        <div className="settings-switch">
          <label>
            <input
              checked={settings.messagePreview}
              onChange={(event) => {
                void updateSetting("messagePreview", event.target.checked);
              }}
              type="checkbox"
            />
            <span>
              <strong>Show message preview</strong>
              <span className="muted">Display the latest message snippet in channel cards.</span>
            </span>
          </label>
        </div>

        <div className="settings-switch">
          <label>
            <input
              checked={settings.typingIndicators}
              onChange={(event) => {
                void updateSetting("typingIndicators", event.target.checked);
              }}
              type="checkbox"
            />
            <span>
              <strong>Typing indicators</strong>
              <span className="muted">Show when other members are typing.</span>
            </span>
          </label>
        </div>

        <div className="settings-switch">
          <label>
            <input
              checked={settings.enterToSend}
              onChange={(event) => {
                void updateSetting("enterToSend", event.target.checked);
              }}
              type="checkbox"
            />
            <span>
              <strong>Enter sends message</strong>
              <span className="muted">Disable this if you prefer multiline input by default.</span>
            </span>
          </label>
        </div>

        <div className="settings-switch">
          <label>
            <input
              checked={settings.compactSidebar}
              onChange={(event) => {
                void updateSetting("compactSidebar", event.target.checked);
              }}
              type="checkbox"
            />
            <span>
              <strong>Compact sidebar</strong>
              <span className="muted">Reduce channel card density for narrower screens.</span>
            </span>
          </label>
        </div>
      </div>

      <div className="settings-field-grid">
        <label className="settings-field">
          <span>Appearance</span>
          <select
            className="message-input"
            onChange={(event) =>
              updateSetting("appearance", event.target.value as AppSettings["appearance"])
            }
            value={settings.appearance}
          >
            <option value="system">System</option>
            <option value="dark">Dark</option>
            <option value="light">Light</option>
          </select>
        </label>

        <label className="settings-field">
          <span>Status</span>
          <select
            className="message-input"
            onChange={(event) =>
              updateSetting("status", event.target.value as AppSettings["status"])
            }
            value={settings.status}
          >
            <option value="online">Online</option>
            <option value="away">Away</option>
            <option value="dnd">Do not disturb</option>
          </select>
        </label>
      </div>
    </section>
  );
}

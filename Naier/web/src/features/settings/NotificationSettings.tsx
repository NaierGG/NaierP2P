import type { Dispatch, SetStateAction } from "react";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import type { AppSettings } from "@/features/settings/settingsStorage";
import {
  ensureDesktopNotificationPermission,
  previewNotificationSound,
} from "@/shared/lib/browserNotifications";

interface NotificationSettingsProps {
  settings: AppSettings;
  setSettings:
    | Dispatch<SetStateAction<AppSettings>>
    | ((next: AppSettings | ((current: AppSettings) => AppSettings)) => void);
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
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <CardTitle>Notifications</CardTitle>
          <CardDescription>
            Tune alerts to keep attention on meaningful activity instead of constant interruption.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-1">
          <SettingRow
            label="Desktop notifications"
            description="Show a browser notification when a message arrives outside the current thread."
            checked={settings.desktopNotifications}
            onChange={(value) => void updateSetting("desktopNotifications", value)}
          />
          <Separator />
          <SettingRow
            label="Sound alerts"
            description="Play a short sound preview when a new message appears."
            checked={settings.soundNotifications}
            onChange={(value) => void updateSetting("soundNotifications", value)}
          />
          <Separator />
          <SettingRow
            label="Message preview"
            description="Show the latest message snippet in the channel list."
            checked={settings.messagePreview}
            onChange={(value) => void updateSetting("messagePreview", value)}
          />
          <Separator />
          <SettingRow
            label="Typing indicators"
            description="Display when other members are composing a message."
            checked={settings.typingIndicators}
            onChange={(value) => void updateSetting("typingIndicators", value)}
          />
          <Separator />
          <SettingRow
            label="Enter sends message"
            description="Turn this off if you prefer Enter to create a new line."
            checked={settings.enterToSend}
            onChange={(value) => void updateSetting("enterToSend", value)}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Interface</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <label className="flex flex-col gap-1.5">
            <span className="text-sm font-medium">Appearance</span>
            <select
              className="flex h-11 w-full rounded-2xl border border-input/80 bg-card/70 px-4 py-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              onChange={(event) =>
                void updateSetting("appearance", event.target.value as AppSettings["appearance"])
              }
              value={settings.appearance}
            >
              <option value="system">System</option>
              <option value="dark">Dark</option>
              <option value="light">Light</option>
            </select>
          </label>

          <label className="flex flex-col gap-1.5">
            <span className="text-sm font-medium">Presence status</span>
            <select
              className="flex h-11 w-full rounded-2xl border border-input/80 bg-card/70 px-4 py-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              onChange={(event) =>
                void updateSetting("status", event.target.value as AppSettings["status"])
              }
              value={settings.status}
            >
              <option value="online">Online</option>
              <option value="away">Away</option>
              <option value="dnd">Do not disturb</option>
            </select>
          </label>
        </CardContent>
      </Card>
    </div>
  );
}

function SettingRow({
  label,
  description,
  checked,
  onChange,
}: {
  label: string;
  description: string;
  checked: boolean;
  onChange: (value: boolean) => void;
}) {
  return (
    <div className="flex items-center justify-between gap-4 py-3">
      <div className="min-w-0">
        <p className="text-sm font-medium">{label}</p>
        <p className="text-xs text-muted-foreground">{description}</p>
      </div>
      <Switch checked={checked} onCheckedChange={onChange} />
    </div>
  );
}

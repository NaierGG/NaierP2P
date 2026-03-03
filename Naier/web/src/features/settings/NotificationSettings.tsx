import type { Dispatch, SetStateAction } from "react";

import type { AppSettings } from "@/features/settings/settingsStorage";
import {
  ensureDesktopNotificationPermission,
  previewNotificationSound,
} from "@/shared/lib/browserNotifications";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import { Separator } from "@/components/ui/separator";

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
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <CardTitle>알림</CardTitle>
          <CardDescription>알림 및 채팅 동작을 설정합니다.</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-1">
          <SettingRow
            label="데스크톱 알림"
            description="새로운 활동에 대해 브라우저 알림을 표시합니다."
            checked={settings.desktopNotifications}
            onChange={(v) => void updateSetting("desktopNotifications", v)}
          />
          <Separator />
          <SettingRow
            label="소리 알림"
            description="새 메시지가 도착하면 소리를 재생합니다."
            checked={settings.soundNotifications}
            onChange={(v) => void updateSetting("soundNotifications", v)}
          />
          <Separator />
          <SettingRow
            label="메시지 미리보기"
            description="채널 카드에 최신 메시지를 미리 표시합니다."
            checked={settings.messagePreview}
            onChange={(v) => void updateSetting("messagePreview", v)}
          />
          <Separator />
          <SettingRow
            label="입력 중 표시"
            description="다른 멤버가 입력 중인 것을 표시합니다."
            checked={settings.typingIndicators}
            onChange={(v) => void updateSetting("typingIndicators", v)}
          />
          <Separator />
          <SettingRow
            label="Enter로 전송"
            description="비활성화하면 여러 줄 입력이 기본됩니다."
            checked={settings.enterToSend}
            onChange={(v) => void updateSetting("enterToSend", v)}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>화면 설정</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <label className="flex flex-col gap-1.5">
            <span className="text-sm font-medium">테마</span>
            <select
              className="flex h-10 w-full rounded-xl border border-input bg-card px-4 py-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              onChange={(e) =>
                void updateSetting("appearance", e.target.value as AppSettings["appearance"])
              }
              value={settings.appearance}
            >
              <option value="system">시스템 설정</option>
              <option value="dark">다크</option>
              <option value="light">라이트</option>
            </select>
          </label>

          <label className="flex flex-col gap-1.5">
            <span className="text-sm font-medium">상태</span>
            <select
              className="flex h-10 w-full rounded-xl border border-input bg-card px-4 py-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              onChange={(e) =>
                void updateSetting("status", e.target.value as AppSettings["status"])
              }
              value={settings.status}
            >
              <option value="online">온라인</option>
              <option value="away">자리 비움</option>
              <option value="dnd">방해 금지</option>
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

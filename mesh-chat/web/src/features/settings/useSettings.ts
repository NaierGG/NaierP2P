import { useEffect, useState } from "react";

import {
  loadSettings,
  saveSettings,
  subscribeSettings,
  type AppSettings,
} from "@/features/settings/settingsStorage";

export function useSettings() {
  const [settings, setSettingsState] = useState<AppSettings>(() => loadSettings());

  useEffect(() => subscribeSettings(setSettingsState), []);

  function setSettings(
    next:
      | AppSettings
      | ((current: AppSettings) => AppSettings)
  ) {
    setSettingsState((current) => {
      const resolved = typeof next === "function" ? next(current) : next;
      saveSettings(resolved);
      return resolved;
    });
  }

  return {
    settings,
    setSettings,
  };
}

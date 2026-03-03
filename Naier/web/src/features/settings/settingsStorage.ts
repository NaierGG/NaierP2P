export interface AppSettings {
  appearance: "system" | "dark" | "light";
  status: "online" | "away" | "dnd";
  enterToSend: boolean;
  desktopNotifications: boolean;
  soundNotifications: boolean;
  messagePreview: boolean;
  typingIndicators: boolean;
  compactSidebar: boolean;
}

const STORAGE_KEY = "naier.settings";
const SETTINGS_EVENT = "naier:settings";

export const defaultSettings: AppSettings = {
  appearance: "system",
  status: "online",
  enterToSend: true,
  desktopNotifications: true,
  soundNotifications: true,
  messagePreview: true,
  typingIndicators: true,
  compactSidebar: false,
};

export function loadSettings(): AppSettings {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return defaultSettings;
    }

    const parsed = JSON.parse(raw) as Partial<AppSettings>;
    return {
      ...defaultSettings,
      ...parsed,
    };
  } catch {
    return defaultSettings;
  }
}

export function saveSettings(settings: AppSettings) {
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
  window.dispatchEvent(
    new CustomEvent<AppSettings>(SETTINGS_EVENT, {
      detail: settings,
    })
  );
}

export function subscribeSettings(listener: (settings: AppSettings) => void) {
  const handleCustomEvent = (event: Event) => {
    const customEvent = event as CustomEvent<AppSettings>;
    listener(customEvent.detail ?? loadSettings());
  };

  const handleStorageEvent = (event: StorageEvent) => {
    if (event.key === STORAGE_KEY) {
      listener(loadSettings());
    }
  };

  window.addEventListener(SETTINGS_EVENT, handleCustomEvent as EventListener);
  window.addEventListener("storage", handleStorageEvent);

  return () => {
    window.removeEventListener(SETTINGS_EVENT, handleCustomEvent as EventListener);
    window.removeEventListener("storage", handleStorageEvent);
  };
}

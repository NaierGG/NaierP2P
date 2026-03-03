import { router } from "@/app/router";
import { store } from "@/app/store";
import { setActiveChannel } from "@/app/store/channelSlice";
import { loadSettings } from "@/features/settings/settingsStorage";
import type { Message } from "@/shared/types";

let audioContext: AudioContext | null = null;
const pendingChannelStorageKey = "naier.pending_channel";

export async function ensureDesktopNotificationPermission(enabled: boolean) {
  if (!enabled || typeof window === "undefined" || !("Notification" in window)) {
    return "unsupported";
  }

  if (Notification.permission === "granted") {
    return "granted";
  }

  return Notification.requestPermission();
}

export async function previewNotificationSound(enabled: boolean) {
  if (!enabled || typeof window === "undefined") {
    return;
  }

  await playNotificationSound();
}

export async function playNotificationSound() {
  if (typeof window === "undefined") {
    return;
  }

  const AudioContextCtor = window.AudioContext || (window as typeof window & { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
  if (!AudioContextCtor) {
    return;
  }

  audioContext = audioContext ?? new AudioContextCtor();
  if (audioContext.state === "suspended") {
    await audioContext.resume();
  }

  const oscillator = audioContext.createOscillator();
  const gain = audioContext.createGain();
  oscillator.type = "sine";
  oscillator.frequency.setValueAtTime(740, audioContext.currentTime);
  gain.gain.setValueAtTime(0.0001, audioContext.currentTime);
  gain.gain.exponentialRampToValueAtTime(0.06, audioContext.currentTime + 0.02);
  gain.gain.exponentialRampToValueAtTime(0.0001, audioContext.currentTime + 0.24);
  oscillator.connect(gain);
  gain.connect(audioContext.destination);
  oscillator.start();
  oscillator.stop(audioContext.currentTime + 0.25);
}

export async function notifyIncomingMessage(message: Message) {
  if (typeof window === "undefined") {
    return;
  }

  const settings = loadSettings();

  if (settings.soundNotifications) {
    await playNotificationSound();
  }

  if (
    settings.desktopNotifications &&
    "Notification" in window &&
    Notification.permission === "granted" &&
    document.visibilityState !== "visible"
  ) {
    const body = settings.messagePreview
      ? message.content || "New message"
      : "Open Naier to view the new message.";

    const notification = new Notification("Naier", {
      body,
      tag: `message-${message.id}`,
    });
    notification.onclick = () => {
      openChannelFromNotification(message.channel_id);
      notification.close();
    };
    window.setTimeout(() => notification.close(), 5000);
  }
}

export function openChannelFromNotification(channelId: string) {
  if (typeof window === "undefined") {
    return;
  }

  window.sessionStorage.setItem(pendingChannelStorageKey, channelId);
  store.dispatch(setActiveChannel(channelId));
  void router.navigate("/app");
  void window.focus();
}

export function consumePendingNotificationChannel() {
  if (typeof window === "undefined") {
    return null;
  }

  const value = window.sessionStorage.getItem(pendingChannelStorageKey);
  if (!value) {
    return null;
  }

  window.sessionStorage.removeItem(pendingChannelStorageKey);
  return value;
}

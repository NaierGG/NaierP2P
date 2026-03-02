import { loadSettings } from "@/features/settings/settingsStorage";
import type { Message } from "@/shared/types";

let audioContext: AudioContext | null = null;

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
      : "Open Mesh Chat to view the new message.";

    const notification = new Notification("Mesh Chat", {
      body,
      tag: `message-${message.id}`,
    });
    window.setTimeout(() => notification.close(), 5000);
  }
}

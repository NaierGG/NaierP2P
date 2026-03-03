export type PresenceStatus = "online" | "away" | "dnd" | "offline";

export interface Presence {
  userId: string;
  status: PresenceStatus;
}

import type { Message } from "@/shared/types/message";
import type { PresenceStatus } from "@/shared/types/presence";
import type { User } from "@/shared/types/user";

export type ClientEventType =
  | "MESSAGE_SEND"
  | "MESSAGE_EDIT"
  | "MESSAGE_DELETE"
  | "TYPING_START"
  | "TYPING_STOP"
  | "REACTION_ADD"
  | "REACTION_REMOVE"
  | "CHANNEL_JOIN"
  | "CHANNEL_LEAVE"
  | "PRESENCE_UPDATE"
  | "READ_ACK";

export type ServerEventType =
  | "MESSAGE_NEW"
  | "MESSAGE_UPDATED"
  | "MESSAGE_DELETED"
  | "TYPING"
  | "REACTION"
  | "PRESENCE"
  | "MEMBER_JOINED"
  | "MEMBER_LEFT"
  | "ERROR";

export interface WSEvent<TPayload = unknown> {
  type: ClientEventType | ServerEventType;
  request_id?: string;
  payload: TPayload;
}

export interface MessageDeletedPayload {
  messageId: string;
  channelId: string;
}

export interface TypingPayload {
  userId: string;
  channelId: string;
  isTyping: boolean;
}

export interface ReactionPayload {
  messageId: string;
  emoji: string;
  userId: string;
  action: "add" | "remove";
}

export interface PresencePayload {
  userId: string;
  status: PresenceStatus;
}

export interface MemberJoinedPayload {
  channelId: string;
  user: User;
}

export interface MemberLeftPayload {
  channelId: string;
  userId: string;
}

export interface ErrorPayload {
  code: string;
  message: string;
}

export type ServerEventPayload =
  | Message
  | MessageDeletedPayload
  | TypingPayload
  | ReactionPayload
  | PresencePayload
  | MemberJoinedPayload
  | MemberLeftPayload
  | ErrorPayload;

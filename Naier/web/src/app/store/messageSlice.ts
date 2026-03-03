import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

import type { Message, PendingMessage } from "@/shared/types";

export interface MessageState {
  messages: Record<string, Message[]>;
  cursors: Record<string, string | null>;
  hasMore: Record<string, boolean>;
  pendingMessages: Record<string, PendingMessage>;
  lastServerEventId: string | null;
}

const initialState: MessageState = {
  messages: {},
  cursors: {},
  hasMore: {},
  pendingMessages: {},
  lastServerEventId: null,
};

const messageSlice = createSlice({
  name: "messages",
  initialState,
  reducers: {
    setMessagesForChannel(
      state,
      action: PayloadAction<{ channelId: string; messages: Message[] }>
    ) {
      state.messages[action.payload.channelId] = action.payload.messages;
    },
    prependMessages(
      state,
      action: PayloadAction<{ channelId: string; messages: Message[] }>
    ) {
      state.messages[action.payload.channelId] = [
        ...action.payload.messages,
        ...(state.messages[action.payload.channelId] ?? []),
      ];
    },
    addMessage(
      state,
      action: PayloadAction<{ channelId: string; message: Message }>
    ) {
      const current = state.messages[action.payload.channelId] ?? [];
      const exists = current.some(
        (message) =>
          message.id === action.payload.message.id ||
          (action.payload.message.client_event_id &&
            message.client_event_id === action.payload.message.client_event_id)
      );
      if (!exists) {
        current.push(action.payload.message);
      } else {
        state.messages[action.payload.channelId] = current.map((message) =>
          message.id === action.payload.message.id ||
          (action.payload.message.client_event_id &&
            message.client_event_id === action.payload.message.client_event_id)
            ? action.payload.message
            : message
        );
        return;
      }
      state.messages[action.payload.channelId] = current;
    },
    reconcilePendingMessage(state, action: PayloadAction<Message>) {
      const pendingEntry = Object.entries(state.pendingMessages).find(
        ([, pending]) =>
          pending.channel_id === action.payload.channel_id &&
          pending.sender_id === action.payload.sender_id &&
          pending.content === action.payload.content
      );

      if (!pendingEntry) {
        return;
      }

      const [clientId, pending] = pendingEntry;
      state.messages[pending.channel_id] = (
        state.messages[pending.channel_id] ?? []
      ).map((message) =>
        "client_id" in message && message.client_id === clientId
          ? action.payload
          : message
      );
      delete state.pendingMessages[clientId];
    },
    updateMessage(
      state,
      action: PayloadAction<{ channelId: string; message: Message }>
    ) {
      state.messages[action.payload.channelId] = (
        state.messages[action.payload.channelId] ?? []
      ).map((message) =>
        message.id === action.payload.message.id ? action.payload.message : message
      );
    },
    removeMessage(
      state,
      action: PayloadAction<{ channelId: string; messageId: string }>
    ) {
      state.messages[action.payload.channelId] = (
        state.messages[action.payload.channelId] ?? []
      ).filter((message) => message.id !== action.payload.messageId);
    },
    markMessageDeleted(
      state,
      action: PayloadAction<{ channelId: string; messageId: string }>
    ) {
      state.messages[action.payload.channelId] = (
        state.messages[action.payload.channelId] ?? []
      ).map((message) =>
        message.id === action.payload.messageId
          ? {
              ...message,
              content: "",
              is_deleted: true,
            }
          : message
      );
    },
    applyReaction(
      state,
      action: PayloadAction<{
        messageId: string;
        emoji: string;
        userId: string;
        action: "add" | "remove";
      }>
    ) {
      for (const channelId of Object.keys(state.messages)) {
        state.messages[channelId] = state.messages[channelId].map((message) => {
          if (message.id !== action.payload.messageId) {
            return message;
          }

          const reactions = [...(message.reactions ?? [])];
          if (action.payload.action === "add") {
            const exists = reactions.some(
              (reaction) =>
                reaction.user_id === action.payload.userId &&
                reaction.emoji === action.payload.emoji
            );
            if (!exists) {
              reactions.push({
                user_id: action.payload.userId,
                emoji: action.payload.emoji,
              });
            }
          } else {
            const nextReactions = reactions.filter(
              (reaction) =>
                !(
                  reaction.user_id === action.payload.userId &&
                  reaction.emoji === action.payload.emoji
                )
            );

            return {
              ...message,
              reactions: nextReactions,
            };
          }

          return {
            ...message,
            reactions,
          };
        });
      }
    },
    setPagination(
      state,
      action: PayloadAction<{
        channelId: string;
        cursor: string | null;
        hasMore: boolean;
      }>
    ) {
      state.cursors[action.payload.channelId] = action.payload.cursor;
      state.hasMore[action.payload.channelId] = action.payload.hasMore;
    },
    addPendingMessage(state, action: PayloadAction<PendingMessage>) {
      state.pendingMessages[action.payload.client_id] = action.payload;
      const channelMessages = state.messages[action.payload.channel_id] ?? [];
      channelMessages.push(action.payload);
      state.messages[action.payload.channel_id] = channelMessages;
    },
    resolvePendingMessage(
      state,
      action: PayloadAction<{ clientId: string; message: Message }>
    ) {
      const pending = state.pendingMessages[action.payload.clientId];
      if (!pending) {
        return;
      }

      state.messages[pending.channel_id] = (
        state.messages[pending.channel_id] ?? []
      ).map((message) =>
        "client_id" in message && message.client_id === action.payload.clientId
          ? action.payload.message
          : message
      );

      delete state.pendingMessages[action.payload.clientId];
    },
    failPendingMessage(
      state,
      action: PayloadAction<{ clientId: string; error: string }>
    ) {
      const pending = state.pendingMessages[action.payload.clientId];
      if (!pending) {
        return;
      }

      pending.status = "failed";
      pending.error = action.payload.error;
    },
    setLastServerEventId(state, action: PayloadAction<string | null>) {
      state.lastServerEventId = action.payload;
    },
  },
});

export const {
  setMessagesForChannel,
  prependMessages,
  addMessage,
  reconcilePendingMessage,
  updateMessage,
  removeMessage,
  markMessageDeleted,
  applyReaction,
  setPagination,
  addPendingMessage,
  resolvePendingMessage,
  failPendingMessage,
  setLastServerEventId,
} = messageSlice.actions;

export default messageSlice.reducer;

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

import type { Channel, Message } from "@/shared/types";

export interface ChannelState {
  channels: Record<string, Channel>;
  activeChannelId: string | null;
  unreadCounts: Record<string, number>;
  lastMessages: Record<string, Message>;
  readStates: Record<string, Record<string, number>>;
}

const initialState: ChannelState = {
  channels: {},
  activeChannelId: null,
  unreadCounts: {},
  lastMessages: {},
  readStates: {},
};

const channelSlice = createSlice({
  name: "channels",
  initialState,
  reducers: {
    setChannels(state, action: PayloadAction<Channel[]>) {
      state.channels = Object.fromEntries(
        action.payload.map((channel) => [channel.id, channel])
      );
    },
    upsertChannel(state, action: PayloadAction<Channel>) {
      state.channels[action.payload.id] = action.payload;
    },
    setActiveChannel(state, action: PayloadAction<string | null>) {
      state.activeChannelId = action.payload;
    },
    setUnreadCount(
      state,
      action: PayloadAction<{ channelId: string; count: number }>
    ) {
      state.unreadCounts[action.payload.channelId] = action.payload.count;
    },
    incrementUnread(state, action: PayloadAction<string>) {
      state.unreadCounts[action.payload] =
        (state.unreadCounts[action.payload] ?? 0) + 1;
    },
    clearUnread(state, action: PayloadAction<string>) {
      state.unreadCounts[action.payload] = 0;
    },
    setLastMessage(
      state,
      action: PayloadAction<{ channelId: string; message: Message }>
    ) {
      state.lastMessages[action.payload.channelId] = action.payload.message;
      if (state.channels[action.payload.channelId]) {
        state.channels[action.payload.channelId] = {
          ...state.channels[action.payload.channelId],
          last_message: action.payload.message,
        };
      }
    },
    setReadState(
      state,
      action: PayloadAction<{
        channelId: string;
        userId: string;
        lastReadSequence: number;
      }>
    ) {
      state.readStates[action.payload.channelId] ??= {};
      state.readStates[action.payload.channelId][action.payload.userId] =
        action.payload.lastReadSequence;
    },
  },
});

export const {
  setChannels,
  upsertChannel,
  setActiveChannel,
  setUnreadCount,
  incrementUnread,
  clearUnread,
  setLastMessage,
  setReadState,
} = channelSlice.actions;

export default channelSlice.reducer;

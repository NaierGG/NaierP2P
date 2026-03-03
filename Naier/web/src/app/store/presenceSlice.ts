import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

import type { PresenceStatus } from "@/shared/types";

export interface PresenceState {
  statuses: Record<string, PresenceStatus>;
  typing: Record<string, string[]>;
}

const initialState: PresenceState = {
  statuses: {},
  typing: {},
};

const presenceSlice = createSlice({
  name: "presence",
  initialState,
  reducers: {
    setPresence(
      state,
      action: PayloadAction<{ userId: string; status: PresenceStatus }>
    ) {
      state.statuses[action.payload.userId] = action.payload.status;
    },
    setTyping(
      state,
      action: PayloadAction<{
        channelId: string;
        userId: string;
        isTyping: boolean;
      }>
    ) {
      const users = new Set(state.typing[action.payload.channelId] ?? []);
      if (action.payload.isTyping) {
        users.add(action.payload.userId);
      } else {
        users.delete(action.payload.userId);
      }
      state.typing[action.payload.channelId] = Array.from(users);
    },
    clearTyping(state, action: PayloadAction<string>) {
      state.typing[action.payload] = [];
    },
  },
});

export const { setPresence, setTyping, clearTyping } = presenceSlice.actions;

export default presenceSlice.reducer;

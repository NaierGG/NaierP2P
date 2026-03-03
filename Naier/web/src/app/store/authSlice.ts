import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

import type { KeyBundle } from "@/shared/lib/crypto";
import type { User } from "@/shared/types";

export interface AuthState {
  user: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  isAuthenticated: boolean;
  keyPair: KeyBundle | null;
}

const initialState: AuthState = {
  user: null,
  accessToken: null,
  refreshToken: null,
  isAuthenticated: false,
  keyPair: null,
};

const authSlice = createSlice({
  name: "auth",
  initialState,
  reducers: {
    setAuth(
      state,
      action: PayloadAction<{
        user: User;
        accessToken: string;
        refreshToken?: string | null;
      }>
    ) {
      state.user = action.payload.user;
      state.accessToken = action.payload.accessToken;
      state.refreshToken = action.payload.refreshToken ?? state.refreshToken;
      state.isAuthenticated = true;
    },
    clearAuth(state) {
      state.user = null;
      state.accessToken = null;
      state.refreshToken = null;
      state.isAuthenticated = false;
      state.keyPair = null;
    },
    setKeyPair(
      state,
      action: PayloadAction<KeyBundle | null>
    ) {
      state.keyPair = action.payload;
    },
    setRefreshToken(state, action: PayloadAction<string | null>) {
      state.refreshToken = action.payload;
    },
  },
});

export const { setAuth, clearAuth, setKeyPair, setRefreshToken } =
  authSlice.actions;

export default authSlice.reducer;

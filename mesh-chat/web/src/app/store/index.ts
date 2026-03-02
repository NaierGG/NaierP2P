import { configureStore } from "@reduxjs/toolkit";
import { setupListeners } from "@reduxjs/toolkit/query";

import { baseApi } from "@/app/store/baseApi";
import authReducer, { clearAuth, setAuth, setRefreshToken } from "@/app/store/authSlice";
import channelReducer from "@/app/store/channelSlice";
import messageReducer from "@/app/store/messageSlice";
import presenceReducer from "@/app/store/presenceSlice";
import { configureAPIClient } from "@/shared/lib/api";
import type { User } from "@/shared/types";

const persistedAccessToken = localStorage.getItem("meshchat.access_token");
const persistedRefreshToken = localStorage.getItem("meshchat.refresh_token");
const persistedUser = localStorage.getItem("meshchat.user");

const preloadedUser = persistedUser ? (JSON.parse(persistedUser) as User) : null;

export const store = configureStore({
  reducer: {
    auth: authReducer,
    channels: channelReducer,
    messages: messageReducer,
    presence: presenceReducer,
    [baseApi.reducerPath]: baseApi.reducer,
  },
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware().concat(baseApi.middleware),
  preloadedState: {
    auth: {
      user: preloadedUser,
      accessToken: persistedAccessToken,
      refreshToken: persistedRefreshToken,
      isAuthenticated: Boolean(persistedAccessToken && preloadedUser),
      keyPair: null,
    },
  },
});

configureAPIClient({
  getTokens: () => {
    const state = store.getState();
    return {
      accessToken: state.auth.accessToken,
      refreshToken: state.auth.refreshToken,
    };
  },
  onTokenUpdate: ({ accessToken, refreshToken }) => {
    const state = store.getState();
    if (!state.auth.user) {
      return;
    }

    store.dispatch(
      setAuth({
        user: state.auth.user,
        accessToken,
        refreshToken,
      })
    );

    localStorage.setItem("meshchat.access_token", accessToken);
    localStorage.setItem("meshchat.refresh_token", refreshToken);
  },
  onAuthFailure: () => {
    localStorage.removeItem("meshchat.access_token");
    localStorage.removeItem("meshchat.refresh_token");
    localStorage.removeItem("meshchat.user");
    store.dispatch(setRefreshToken(null));
    store.dispatch(clearAuth());
  },
});

store.subscribe(() => {
  const state = store.getState();

  if (state.auth.accessToken) {
    localStorage.setItem("meshchat.access_token", state.auth.accessToken);
  } else {
    localStorage.removeItem("meshchat.access_token");
  }

  if (state.auth.refreshToken) {
    localStorage.setItem("meshchat.refresh_token", state.auth.refreshToken);
  } else {
    localStorage.removeItem("meshchat.refresh_token");
  }

  if (state.auth.user) {
    localStorage.setItem("meshchat.user", JSON.stringify(state.auth.user));
  } else {
    localStorage.removeItem("meshchat.user");
  }
});

setupListeners(store.dispatch);

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;

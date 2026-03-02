import { api } from "@/shared/lib/api";
import {
  isLikelyNetworkError,
  mockLogin,
  mockRefresh,
  mockRegister,
  mockRequestChallenge,
} from "@/shared/lib/mockApi";
import type { User } from "@/shared/types";

interface ChallengeResponse {
  challenge: string;
  ttl: number;
}

interface AuthResponse {
  access_token: string;
  refresh_token: string;
  user: User;
}

export async function requestChallenge(username: string) {
  try {
    const response = await api.post<ChallengeResponse>("/auth/challenge", {
      username,
    });

    return response.data;
  } catch (error) {
    if (!isLikelyNetworkError(error)) {
      throw error;
    }

    return mockRequestChallenge(username);
  }
}

export async function registerWithKeyPair(payload: {
  username: string;
  displayName: string;
  publicKey: string;
  signature: string;
}) {
  try {
    const response = await api.post<AuthResponse>("/auth/register", {
      username: payload.username,
      display_name: payload.displayName,
      public_key: payload.publicKey,
      signature: payload.signature,
    });

    return response.data;
  } catch (error) {
    if (!isLikelyNetworkError(error)) {
      throw error;
    }

    return mockRegister(payload);
  }
}

export async function loginWithChallenge(payload: {
  username: string;
  challenge: string;
  signature: string;
}) {
  try {
    const response = await api.post<AuthResponse>("/auth/login", {
      username: payload.username,
      challenge: payload.challenge,
      signature: payload.signature,
    });

    return response.data;
  } catch (error) {
    if (!isLikelyNetworkError(error)) {
      throw error;
    }

    return mockLogin(payload);
  }
}

export async function refreshAuth(refreshToken: string) {
  try {
    const response = await api.post<AuthResponse>("/auth/refresh", {
      refresh_token: refreshToken,
    });

    return response.data;
  } catch (error) {
    if (!isLikelyNetworkError(error)) {
      throw error;
    }

    return mockRefresh(refreshToken);
  }
}

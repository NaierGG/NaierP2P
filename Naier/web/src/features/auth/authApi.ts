import { api } from "@/shared/lib/api";
import {
  mockLogin,
  mockRefresh,
  mockRegister,
  mockRequestChallenge,
  shouldUseMockFallback,
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

export async function requestChallenge(payload: {
  username: string;
  deviceSigningKey?: string;
  deviceName?: string;
  platform?: "web";
}) {
  try {
    const response = await api.post<ChallengeResponse>("/auth/challenge", {
      username: payload.username,
      device_signing_key: payload.deviceSigningKey,
      device_name: payload.deviceName,
      platform: payload.platform,
    });

    return response.data;
  } catch (error) {
    if (!shouldUseMockFallback(error)) {
      throw error;
    }

    return mockRequestChallenge(payload.username);
  }
}

export async function registerWithKeyPair(payload: {
  username: string;
  displayName: string;
  identitySigningKey: string;
  identityExchangeKey: string;
  deviceSigningKey: string;
  deviceExchangeKey: string;
  deviceSignature: string;
  identitySignatureOverDevice: string;
  deviceName: string;
  platform: "web";
  inviteCode?: string;
}) {
  try {
    const response = await api.post<AuthResponse>("/auth/register", {
      username: payload.username,
      display_name: payload.displayName,
      public_key: payload.identitySigningKey,
      identity_signing_key: payload.identitySigningKey,
      identity_exchange_key: payload.identityExchangeKey,
      device_signing_key: payload.deviceSigningKey,
      device_exchange_key: payload.deviceExchangeKey,
      device_signature: payload.deviceSignature,
      identity_signature_over_device: payload.identitySignatureOverDevice,
      device_name: payload.deviceName,
      platform: payload.platform,
      invite_code: payload.inviteCode,
    });

    return response.data;
  } catch (error) {
    if (!shouldUseMockFallback(error)) {
      throw error;
    }

    return mockRegister({
      username: payload.username,
      displayName: payload.displayName,
      publicKey: payload.identitySigningKey,
      signature: payload.deviceSignature,
      inviteCode: payload.inviteCode,
    });
  }
}

export async function loginWithChallenge(payload: {
  username: string;
  challenge: string;
  signature: string;
  deviceSigningKey: string;
  deviceName: string;
  platform: "web";
}) {
  try {
    const response = await api.post<AuthResponse>("/auth/login", {
      username: payload.username,
      challenge: payload.challenge,
      device_signature: payload.signature,
      device_signing_key: payload.deviceSigningKey,
      device_name: payload.deviceName,
      platform: payload.platform,
    });

    return response.data;
  } catch (error) {
    if (!shouldUseMockFallback(error)) {
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
    if (!shouldUseMockFallback(error)) {
      throw error;
    }

    return mockRefresh(refreshToken);
  }
}

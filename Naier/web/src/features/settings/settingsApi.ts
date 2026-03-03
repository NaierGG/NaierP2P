import { api } from "@/shared/lib/api";
import {
  isLikelyNetworkError,
  mockApproveDevice,
  mockCreatePendingDevice,
  mockGetProfile,
  mockListDevices,
  mockRevokeDevice,
  mockUpdateProfile,
} from "@/shared/lib/mockApi";
import type { Device, User } from "@/shared/types";

interface ProfileResponse {
  user: User;
}

interface DevicesResponse {
  devices: Array<Device & { current?: boolean }>;
}

interface DeviceResponse {
  device: Device & { current?: boolean };
}

export async function fetchProfile(accessToken: string | null) {
  try {
    const response = await api.get<ProfileResponse>("/auth/me");
    return response.data;
  } catch (error) {
    if (!isLikelyNetworkError(error) || !accessToken) {
      throw error;
    }

    return mockGetProfile(accessToken);
  }
}

export async function updateProfile(
  accessToken: string | null,
  payload: { display_name: string; bio?: string; avatar_url?: string }
) {
  try {
    const response = await api.put<ProfileResponse>("/auth/me", payload);
    return response.data;
  } catch (error) {
    if (!isLikelyNetworkError(error) || !accessToken) {
      throw error;
    }

    return mockUpdateProfile(accessToken, payload);
  }
}

export async function fetchDevices(accessToken: string | null) {
  try {
    const response = await api.get<DevicesResponse>("/auth/devices");
    return response.data;
  } catch (error) {
    if (!isLikelyNetworkError(error) || !accessToken) {
      throw error;
    }

    return mockListDevices(accessToken);
  }
}

export async function revokeDevice(accessToken: string | null, deviceId: string) {
  try {
    await api.delete(`/auth/devices/${deviceId}`);
  } catch (error) {
    if (!isLikelyNetworkError(error) || !accessToken) {
      throw error;
    }

    await mockRevokeDevice(accessToken, deviceId);
  }
}

export async function createPendingDevice(
  accessToken: string | null,
  payload: {
    device_signing_key: string;
    device_exchange_key: string;
    device_name: string;
    platform: "web" | "ios" | "android";
  }
) {
  try {
    const response = await api.post<DeviceResponse>("/auth/devices/pending", payload);
    return response.data;
  } catch (error) {
    if (!isLikelyNetworkError(error) || !accessToken) {
      throw error;
    }

    return mockCreatePendingDevice(accessToken, payload);
  }
}

export async function approveDevice(accessToken: string | null, deviceId: string) {
  try {
    await api.post("/auth/devices/approve", {
      device_id: deviceId,
    });
  } catch (error) {
    if (!isLikelyNetworkError(error) || !accessToken) {
      throw error;
    }

    await mockApproveDevice(accessToken, deviceId);
  }
}

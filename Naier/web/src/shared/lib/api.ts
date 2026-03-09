import axios, { type AxiosError, type InternalAxiosRequestConfig } from "axios";
import { API_BASE_URL } from "@/shared/lib/runtime";

interface Tokens {
  accessToken: string | null;
  refreshToken: string | null;
}

type TokenProvider = () => Tokens;
type TokenUpdateHandler = (tokens: {
  accessToken: string;
  refreshToken: string;
}) => void;
type AuthFailureHandler = () => void;

declare module "axios" {
  interface InternalAxiosRequestConfig {
    _retry?: boolean;
  }
}

let getTokens: TokenProvider = () => ({
  accessToken: null,
  refreshToken: null,
});
let onTokenUpdate: TokenUpdateHandler = () => undefined;
let onAuthFailure: AuthFailureHandler = () => undefined;
let refreshPromise: Promise<{ accessToken: string; refreshToken: string } | null> | null =
  null;

export const api = axios.create({
  baseURL: API_BASE_URL,
  withCredentials: false,
});

export function configureAPIClient(options: {
  getTokens: TokenProvider;
  onTokenUpdate: TokenUpdateHandler;
  onAuthFailure: AuthFailureHandler;
}) {
  getTokens = options.getTokens;
  onTokenUpdate = options.onTokenUpdate;
  onAuthFailure = options.onAuthFailure;
}

api.interceptors.request.use((config) => attachToken(config));

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config;
    if (!originalRequest || error.response?.status !== 401 || originalRequest._retry) {
      return Promise.reject(error);
    }

    originalRequest._retry = true;
    const refreshed = await refreshAccessToken();

    if (!refreshed) {
      onAuthFailure();
      return Promise.reject(error);
    }

    originalRequest.headers = originalRequest.headers ?? {};
    originalRequest.headers.Authorization = `Bearer ${refreshed.accessToken}`;

    return api(originalRequest);
  }
);

function attachToken(config: InternalAxiosRequestConfig) {
  const { accessToken } = getTokens();
  if (accessToken) {
    config.headers = config.headers ?? {};
    config.headers.Authorization = `Bearer ${accessToken}`;
  }

  return config;
}

async function refreshAccessToken() {
  if (!refreshPromise) {
    refreshPromise = (async () => {
      const { refreshToken } = getTokens();
      if (!refreshToken) {
        return null;
      }

      try {
        const response = await axios.post<{ access_token: string; refresh_token: string }>(
          `${API_BASE_URL}/auth/refresh`,
          { refresh_token: refreshToken }
        );

        const nextTokens = {
          accessToken: response.data.access_token,
          refreshToken: response.data.refresh_token,
        };

        onTokenUpdate(nextTokens);
        return nextTokens;
      } catch {
        return null;
      } finally {
        refreshPromise = null;
      }
    })();
  }

  return refreshPromise;
}

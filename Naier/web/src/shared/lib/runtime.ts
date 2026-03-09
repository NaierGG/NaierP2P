const rawAPIBaseURL = import.meta.env.VITE_API_BASE_URL as string | undefined;
const rawWSURL = import.meta.env.VITE_WS_URL as string | undefined;
const rawMockFallback = import.meta.env.VITE_ENABLE_MOCK_FALLBACK as string | undefined;

export const API_BASE_URL = (() => {
  if (rawAPIBaseURL) {
    return rawAPIBaseURL;
  }
  if (import.meta.env.PROD) {
    throw new Error("VITE_API_BASE_URL is required in production builds.");
  }
  return "http://localhost:8080/api/v1";
})();

export const WS_URL = rawWSURL;

export function isMockFallbackEnabled() {
  if (rawMockFallback != null) {
    return rawMockFallback === "true";
  }

  return import.meta.env.DEV;
}

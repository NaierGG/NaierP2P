import { Suspense, lazy } from "react";
import { useSelector } from "react-redux";
import { Navigate, Outlet, createBrowserRouter } from "react-router-dom";
import { Loader2 } from "lucide-react";

import type { RootState } from "@/app/store";

const LoginPage = lazy(() => import("@/features/auth/LoginPage"));
const RegisterPage = lazy(() => import("@/features/auth/RegisterPage"));
const KeygenFlow = lazy(() => import("@/features/auth/KeygenFlow"));
const DeviceLinkPage = lazy(() => import("@/features/auth/DeviceLinkPage"));
const ChannelList = lazy(() => import("@/features/channels/ChannelList"));
const SettingsPage = lazy(() => import("@/features/settings/SettingsPage"));

function AuthLayout() {
  return (
    <div className="grid min-h-screen place-items-center bg-background p-6">
      <Suspense fallback={<RouteFallback />}>
        <Outlet />
      </Suspense>
    </div>
  );
}

function ProtectedLayout() {
  const isAuthenticated = useSelector(
    (state: RootState) => state.auth.isAuthenticated
  );
  if (!isAuthenticated) {
    return <Navigate replace to="/auth/login" />;
  }

  return (
    <Suspense fallback={<RouteFallback />}>
      <Outlet />
    </Suspense>
  );
}

function RouteFallback() {
  return (
    <div className="grid min-h-screen place-items-center bg-background">
      <div className="flex flex-col items-center gap-3 text-muted-foreground">
        <Loader2 className="h-8 w-8 animate-spin" />
        <p className="text-sm">로딩 중...</p>
      </div>
    </div>
  );
}

export const router = createBrowserRouter([
  {
    path: "/",
    element: <Navigate replace to="/app" />,
  },
  {
    path: "/auth",
    element: <AuthLayout />,
    children: [
      {
        index: true,
        element: <Navigate replace to="/auth/login" />,
      },
      {
        path: "login",
        element: <LoginPage />,
      },
      {
        path: "register",
        element: <RegisterPage />,
      },
      {
        path: "keygen",
        element: <KeygenFlow />,
      },
      {
        path: "device",
        element: <DeviceLinkPage />,
      },
    ],
  },
  {
    path: "/app",
    element: <ProtectedLayout />,
    children: [
      {
        index: true,
        element: <ChannelList />,
      },
      {
        path: "settings",
        element: <SettingsPage />,
      },
    ],
  },
]);

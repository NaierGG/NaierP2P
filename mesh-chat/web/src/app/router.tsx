import { Suspense, lazy } from "react";
import { useSelector } from "react-redux";
import { Navigate, Outlet, createBrowserRouter } from "react-router-dom";

import type { RootState } from "@/app/store";

const LoginPage = lazy(() => import("@/features/auth/LoginPage"));
const RegisterPage = lazy(() => import("@/features/auth/RegisterPage"));
const KeygenFlow = lazy(() => import("@/features/auth/KeygenFlow"));
const ChannelList = lazy(() => import("@/features/channels/ChannelList"));
const SettingsPage = lazy(() => import("@/features/settings/SettingsPage"));

function AuthLayout() {
  return (
    <div className="auth-layout">
      <Suspense fallback={<RouteFallback label="Loading auth" />}>
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
    <Suspense fallback={<RouteFallback label="Loading app" />}>
      <Outlet />
    </Suspense>
  );
}

function RouteFallback({ label }: { label: string }) {
  return (
    <div className="route-fallback">
      <div className="panel">
        <p className="eyebrow">{label}</p>
        <h2>Preparing interface</h2>
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

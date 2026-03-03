import { useEffect } from "react";
import { RouterProvider } from "react-router-dom";

import { router } from "@/app/router";
import { useSettings } from "@/features/settings/useSettings";

export default function App() {
  const { settings } = useSettings();

  useEffect(() => {
    const root = document.documentElement;
    const systemPrefersLight = window.matchMedia("(prefers-color-scheme: light)").matches;
    const resolvedTheme =
      settings.appearance === "system"
        ? systemPrefersLight
          ? "light"
          : "dark"
        : settings.appearance;

    root.dataset.theme = resolvedTheme;
  }, [settings.appearance]);

  return <RouterProvider router={router} />;
}

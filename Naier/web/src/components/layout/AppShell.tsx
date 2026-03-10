import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface AppShellProps {
  sidebar: ReactNode;
  children: ReactNode;
  className?: string;
}

export default function AppShell({ sidebar, children, className }: AppShellProps) {
  return (
    <div
      className={cn(
        "app-noise relative flex h-screen w-full overflow-hidden bg-transparent p-3 md:p-4 lg:p-5",
        className
      )}
    >
      <div className="relative z-10 flex w-full min-w-0 gap-3">
        {sidebar}
        {children}
      </div>
    </div>
  );
}

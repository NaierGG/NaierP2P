import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface AppShellProps {
  sidebar: ReactNode;
  children: ReactNode;
  className?: string;
}

export default function AppShell({ sidebar, children, className }: AppShellProps) {
  return (
    <div className={cn("flex h-screen w-full overflow-hidden bg-background", className)}>
      {sidebar}
      {children}
    </div>
  );
}

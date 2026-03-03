import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface SidebarProps {
  children: ReactNode;
  className?: string;
}

export default function Sidebar({ children, className }: SidebarProps) {
  return (
    <aside
      className={cn(
        "flex w-[300px] min-w-[260px] max-w-[340px] flex-col border-r border-sidebar-border bg-sidebar",
        className
      )}
    >
      {children}
    </aside>
  );
}

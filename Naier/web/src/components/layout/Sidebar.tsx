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
        "flex w-[312px] min-w-[270px] max-w-[352px] flex-col overflow-hidden rounded-[1.75rem] border border-sidebar-border/80 bg-sidebar/90 shadow-[0_24px_60px_rgba(5,12,24,0.42)] backdrop-blur-xl",
        className
      )}
    >
      {children}
    </aside>
  );
}

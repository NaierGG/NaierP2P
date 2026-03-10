import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface ChatPanelProps {
  children: ReactNode;
  className?: string;
}

export default function ChatPanel({ children, className }: ChatPanelProps) {
  return (
    <main
      className={cn(
        "flex min-w-0 flex-1 flex-col overflow-hidden rounded-[1.9rem] border border-border/70 bg-card/85 shadow-[0_28px_80px_rgba(3,10,22,0.38)] backdrop-blur-xl",
        className
      )}
    >
      {children}
    </main>
  );
}

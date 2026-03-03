import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface ChatPanelProps {
  children: ReactNode;
  className?: string;
}

export default function ChatPanel({ children, className }: ChatPanelProps) {
  return (
    <main className={cn("flex min-w-0 flex-1 flex-col", className)}>
      {children}
    </main>
  );
}

import { Hash, Lock, Users } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { Channel } from "@/shared/types";

type ConnectionState = "connecting" | "connected" | "disconnected" | string;

interface ChatHeaderProps {
  channel: Channel | null;
  title?: string;
  meta?: string;
  connectionState: ConnectionState;
  isMockMode: boolean;
}

const statusConfig: Record<
  string,
  { label: string; variant: "warning" | "success" | "destructive" }
> = {
  connecting: { label: "Connecting", variant: "warning" },
  connected: { label: "Secure session", variant: "success" },
  disconnected: { label: "Offline", variant: "destructive" },
};

export default function ChatHeader({
  channel,
  title,
  meta,
  connectionState,
  isMockMode,
}: ChatHeaderProps) {
  const status = statusConfig[connectionState] ?? statusConfig.connecting;

  return (
    <header className="flex items-center justify-between border-b border-border/70 bg-card/55 px-6 py-5 backdrop-blur-sm">
      <div className="flex min-w-0 items-center gap-3">
        {channel ? (
          <>
            <div className="flex h-11 w-11 items-center justify-center rounded-2xl border border-primary/15 bg-primary/10 text-primary shadow-[0_12px_24px_rgba(48,162,198,0.16)]">
              {channel.is_encrypted ? <Lock className="h-4 w-4" /> : <Hash className="h-4 w-4" />}
            </div>
            <div className="min-w-0">
              <p className="mb-1 text-[10px] font-semibold uppercase tracking-[0.24em] text-primary/75">
                Active conversation
              </p>
              <h1 className="truncate text-lg font-semibold">{title ?? channel.name}</h1>
              <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <Users className="h-3 w-3" />
                <span>{meta ?? `${channel.member_count} members`}</span>
              </p>
            </div>
          </>
        ) : (
          <div>
            <p className="mb-1 text-[10px] font-semibold uppercase tracking-[0.24em] text-primary/75">
              Inbox
            </p>
            <h1 className="text-lg font-semibold text-foreground">Select a conversation</h1>
            <p className="text-xs text-muted-foreground">
              Choose a channel on the left to open the encrypted thread.
            </p>
          </div>
        )}
      </div>

      <div className="flex items-center gap-2">
        {isMockMode && (
          <Badge variant="warning" className="text-xs">
            Mock mode
          </Badge>
        )}
        <Badge
          variant={status.variant}
          className={cn("text-xs", connectionState === "connecting" && "animate-pulse")}
        >
          <span
            className={cn(
              "mr-1.5 inline-block h-1.5 w-1.5 rounded-full",
              status.variant === "success" && "bg-emerald-500",
              status.variant === "warning" && "bg-amber-500",
              status.variant === "destructive" && "bg-red-500"
            )}
          />
          {status.label}
        </Badge>
      </div>
    </header>
  );
}

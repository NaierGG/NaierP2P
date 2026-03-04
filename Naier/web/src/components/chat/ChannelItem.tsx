import { Hash, Lock } from "lucide-react";

import { cn } from "@/lib/utils";
import type { Channel } from "@/shared/types";

interface ChannelItemProps {
  channel: Channel;
  title: string;
  preview: string;
  isActive: boolean;
  onClick: () => void;
}

export default function ChannelItem({
  channel,
  title,
  preview,
  isActive,
  onClick,
}: ChannelItemProps) {
  const lastTime = channel.last_message?.created_at ?? channel.created_at;

  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "group flex w-full items-start gap-3 rounded-xl px-3 py-3 text-left transition-colors",
        isActive ? "bg-primary/10 text-foreground" : "text-foreground/80 hover:bg-accent"
      )}
    >
      <div
        className={cn(
          "mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg transition-colors",
          isActive
            ? "bg-primary/20 text-primary"
            : "bg-muted text-muted-foreground group-hover:bg-accent"
        )}
      >
        {channel.is_encrypted ? <Lock className="h-3.5 w-3.5" /> : <Hash className="h-3.5 w-3.5" />}
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center justify-between gap-2">
          <span className="truncate text-sm font-medium">{title}</span>
          <span className="shrink-0 text-[11px] text-muted-foreground">
            {formatRelativeTime(lastTime)}
          </span>
        </div>
        <p className="mt-0.5 truncate text-xs text-muted-foreground">{preview}</p>
      </div>
    </button>
  );
}

function formatRelativeTime(iso: string): string {
  const date = new Date(iso);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMin = Math.floor(diffMs / 60000);

  if (diffMin < 1) return "now";
  if (diffMin < 60) return `${diffMin}m`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h`;
  const diffDay = Math.floor(diffHr / 24);
  if (diffDay < 7) return `${diffDay}d`;
  return date.toLocaleDateString("ko-KR", { month: "short", day: "numeric" });
}

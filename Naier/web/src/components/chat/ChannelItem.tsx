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
        "group flex w-full items-start gap-3 rounded-[1.35rem] border px-3.5 py-3.5 text-left transition-all duration-200",
        isActive
          ? "border-primary/20 bg-primary/10 text-foreground shadow-[0_16px_34px_rgba(48,162,198,0.15)]"
          : "border-transparent bg-transparent text-foreground/80 hover:border-border/60 hover:bg-card/60 hover:text-foreground"
      )}
    >
      <div
        className={cn(
          "mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl border transition-colors",
          isActive
            ? "border-primary/15 bg-primary/15 text-primary"
            : "border-border/70 bg-secondary/60 text-muted-foreground group-hover:border-border group-hover:bg-accent"
        )}
      >
        {channel.is_encrypted ? <Lock className="h-3.5 w-3.5" /> : <Hash className="h-3.5 w-3.5" />}
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center justify-between gap-2">
          <span className="truncate text-sm font-semibold">{title}</span>
          <span className="shrink-0 text-[11px] text-muted-foreground/90">
            {formatRelativeTime(lastTime)}
          </span>
        </div>
        <p className="mt-1 line-clamp-2 text-xs leading-5 text-muted-foreground">{preview}</p>
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
  return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

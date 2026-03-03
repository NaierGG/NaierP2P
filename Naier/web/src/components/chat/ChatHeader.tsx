import { Hash, Lock, Users } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { Channel } from "@/shared/types";

type ConnectionState = "connecting" | "connected" | "disconnected" | string;

interface ChatHeaderProps {
  channel: Channel | null;
  connectionState: ConnectionState;
  isMockMode: boolean;
}

const statusConfig: Record<string, { label: string; variant: "warning" | "success" | "destructive" }> = {
  connecting: { label: "연결 중...", variant: "warning" },
  connected: { label: "보안 연결됨", variant: "success" },
  disconnected: { label: "연결 끊김", variant: "destructive" },
};

export default function ChatHeader({ channel, connectionState, isMockMode }: ChatHeaderProps) {
  const status = statusConfig[connectionState] ?? statusConfig.connecting;

  return (
    <header className="flex items-center justify-between border-b border-border px-6 py-4">
      <div className="flex items-center gap-3 min-w-0">
        {channel ? (
          <>
            <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-primary/10 text-primary">
              {channel.is_encrypted ? <Lock className="h-4 w-4" /> : <Hash className="h-4 w-4" />}
            </div>
            <div className="min-w-0">
              <h1 className="truncate text-base font-semibold">{channel.name}</h1>
              <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <Users className="h-3 w-3" />
                <span>{channel.member_count}명</span>
                <span className="text-border">·</span>
                <span>{channel.type === "dm" ? "DM" : channel.type === "group" ? "그룹" : "공개"}</span>
              </p>
            </div>
          </>
        ) : (
          <div>
            <h1 className="text-base font-semibold text-muted-foreground">채널을 선택하세요</h1>
          </div>
        )}
      </div>

      <div className="flex items-center gap-2">
        {isMockMode && (
          <Badge variant="warning" className="text-xs">오프라인 모드</Badge>
        )}
        <Badge
          variant={status.variant}
          className={cn(
            "text-xs",
            connectionState === "connecting" && "animate-pulse"
          )}
        >
          <span className={cn(
            "mr-1.5 inline-block h-1.5 w-1.5 rounded-full",
            status.variant === "success" && "bg-emerald-500",
            status.variant === "warning" && "bg-amber-500",
            status.variant === "destructive" && "bg-red-500",
          )} />
          {status.label}
        </Badge>
      </div>
    </header>
  );
}

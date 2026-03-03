import { Check, CheckCheck, AlertCircle, Clock } from "lucide-react";
import { cn } from "@/lib/utils";
import type { Message } from "@/shared/types";

interface MessageBubbleProps {
  message: Message;
  isOwn: boolean;
}

export default function MessageBubble({ message, isOwn }: MessageBubbleProps) {
  return (
    <div className={cn("flex w-full px-2", isOwn ? "justify-end" : "justify-start")}>
      <div
        className={cn(
          "max-w-[70%] rounded-2xl px-4 py-2.5",
          isOwn
            ? "rounded-br-md bg-primary/15 text-foreground"
            : "rounded-bl-md bg-bubble text-foreground"
        )}
      >
        {!isOwn && (
          <p className="mb-0.5 text-xs font-medium text-primary">
            {message.sender_id.slice(0, 8)}
          </p>
        )}

        <div className="text-sm leading-relaxed whitespace-pre-wrap break-words">
          {message.is_deleted ? (
            <span className="italic text-muted-foreground">삭제된 메시지</span>
          ) : (
            message.content
          )}
        </div>

        {message.reactions && message.reactions.length > 0 && (
          <div className="mt-1.5 flex flex-wrap gap-1">
            {message.reactions.map((reaction, i) => (
              <span
                key={`${reaction.user_id}-${reaction.emoji}-${i}`}
                className="inline-flex items-center rounded-full bg-accent px-1.5 py-0.5 text-xs"
              >
                {reaction.emoji}
              </span>
            ))}
          </div>
        )}

        <div className={cn(
          "mt-1 flex items-center gap-1.5 text-[11px] text-muted-foreground",
          isOwn && "justify-end"
        )}>
          <span>{new Date(message.created_at).toLocaleTimeString("ko-KR", { hour: "2-digit", minute: "2-digit" })}</span>
          {isOwn && <StatusIcon status={message.status} />}
        </div>
      </div>
    </div>
  );
}

function StatusIcon({ status }: { status?: Message["status"] }) {
  switch (status) {
    case "sending":
      return <Clock className="h-3 w-3 text-muted-foreground" />;
    case "sent":
      return <CheckCheck className="h-3 w-3 text-primary" />;
    case "failed":
      return <AlertCircle className="h-3 w-3 text-destructive" />;
    default:
      return <Check className="h-3 w-3 text-muted-foreground" />;
  }
}

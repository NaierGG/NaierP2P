import { AlertCircle, Check, CheckCheck, Clock } from "lucide-react";

import { cn } from "@/lib/utils";
import type { Message } from "@/shared/types";

interface MessageBubbleProps {
  message: Message;
  isOwn: boolean;
  senderLabel?: string;
}

export default function MessageBubble({ message, isOwn, senderLabel }: MessageBubbleProps) {
  return (
    <div
      data-testid="chat-message-bubble"
      className={cn("flex w-full px-2", isOwn ? "justify-end" : "justify-start")}
    >
      <div
        className={cn(
          "max-w-[76%] rounded-[1.4rem] border px-4 py-3 shadow-[0_12px_28px_rgba(4,10,20,0.14)]",
          isOwn
            ? "rounded-br-md border-primary/15 bg-bubble-own text-foreground"
            : "rounded-bl-md border-border/70 bg-bubble text-foreground"
        )}
      >
        {!isOwn && (
          <p className="mb-1 text-xs font-semibold uppercase tracking-[0.14em] text-primary/85">
            {senderLabel ?? message.sender_id.slice(0, 8)}
          </p>
        )}

        <div className="whitespace-pre-wrap break-words text-sm leading-7">
          {message.is_deleted ? (
            <span className="italic text-muted-foreground">Deleted message</span>
          ) : (
            message.content
          )}
        </div>

        {message.reactions && message.reactions.length > 0 && (
          <div className="mt-2 flex flex-wrap gap-1.5">
            {message.reactions.map((reaction, index) => (
              <span
                key={`${reaction.user_id}-${reaction.emoji}-${index}`}
                className="inline-flex items-center rounded-full border border-border/70 bg-accent/70 px-2 py-0.5 text-xs"
              >
                {reaction.emoji}
              </span>
            ))}
          </div>
        )}

        <div
          className={cn(
            "mt-2 flex items-center gap-1.5 text-[11px] text-muted-foreground",
            isOwn && "justify-end"
          )}
        >
          <span>
            {new Date(message.created_at).toLocaleTimeString("en-US", {
              hour: "2-digit",
              minute: "2-digit",
            })}
          </span>
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

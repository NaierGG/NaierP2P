import type { Message } from "@/shared/types";

interface MessageBubbleProps {
  message: Message;
  isOwn: boolean;
}

export default function MessageBubble({
  message,
  isOwn,
}: MessageBubbleProps) {
  return (
    <article className={`message-row ${isOwn ? "is-own" : ""}`}>
      <div className={`message-bubble ${isOwn ? "is-own" : ""}`}>
        <div className="message-meta">
          <span>{isOwn ? "You" : message.sender_id.slice(0, 8)}</span>
          <span>{new Date(message.created_at).toLocaleTimeString()}</span>
        </div>
        <div className="message-content">
          {message.is_deleted ? (
            <em className="muted">Message deleted</em>
          ) : (
            message.content
          )}
        </div>
        {message.reactions && message.reactions.length > 0 ? (
          <div className="reaction-strip">
            {message.reactions.map((reaction, index) => (
              <span className="reaction-pill" key={`${reaction.user_id}-${reaction.emoji}-${index}`}>
                {reaction.emoji}
              </span>
            ))}
          </div>
        ) : null}
        {message.status ? (
          <div className="message-status">{message.status}</div>
        ) : null}
      </div>
    </article>
  );
}

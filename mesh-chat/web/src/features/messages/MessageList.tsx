import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";

import { useAppDispatch, useAppSelector } from "@/app/store/hooks";
import { setPagination, setMessagesForChannel, prependMessages } from "@/app/store/messageSlice";
import { api } from "@/shared/lib/api";
import { isLikelyNetworkError, mockListMessages } from "@/shared/lib/mockApi";
import type { Message } from "@/shared/types";
import MessageBubble from "@/features/messages/MessageBubble";
import TypingIndicator from "@/features/presence/TypingIndicator";

interface MessageListProps {
  channelId: string | null;
}

interface MessageListResponse {
  messages: Message[];
  next_cursor?: string;
  has_more: boolean;
}

export default function MessageList({ channelId }: MessageListProps) {
  const dispatch = useAppDispatch();
  const currentUserId = useAppSelector((state) => state.auth.user?.id ?? null);
  const messages = useAppSelector((state) =>
    channelId ? state.messages.messages[channelId] ?? [] : []
  );
  const cursor = useAppSelector((state) =>
    channelId ? state.messages.cursors[channelId] ?? null : null
  );
  const hasMore = useAppSelector((state) =>
    channelId ? state.messages.hasMore[channelId] ?? true : false
  );
  const typingUsers = useAppSelector((state) =>
    channelId ? state.presence.typing[channelId] ?? [] : []
  );

  const parentRef = useRef<HTMLDivElement | null>(null);
  const [loading, setLoading] = useState(false);
  const [showJumpToLatest, setShowJumpToLatest] = useState(false);

  const rows = useMemo(
    () =>
      messages.map((message, index) => ({
        key: message.id || `message-${index}`,
        message,
      })),
    [messages]
  );

  useEffect(() => {
    if (!channelId) {
      return;
    }

    let cancelled = false;

    const loadInitial = async () => {
      setLoading(true);
      try {
        let response: MessageListResponse;
        try {
          const remote = await api.get<MessageListResponse>(
            `/channels/${channelId}/messages`,
            { params: { limit: 40 } }
          );
          response = remote.data;
        } catch (error) {
          if (!isLikelyNetworkError(error)) {
            throw error;
          }

          response = await mockListMessages({
            channelId,
            limit: 40,
          });
        }
        if (cancelled) {
          return;
        }

        const ordered = [...response.messages].reverse();
        dispatch(setMessagesForChannel({ channelId, messages: ordered }));
        dispatch(
          setPagination({
            channelId,
            cursor: response.next_cursor ?? null,
            hasMore: response.has_more,
          })
        );
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    void loadInitial();

    return () => {
      cancelled = true;
    };
  }, [channelId, dispatch]);

  useEffect(() => {
    if (!channelId || !hasMore || !cursor || !parentRef.current || loading) {
      return;
    }

    const element = parentRef.current;
    const onScroll = async () => {
      if (element.scrollTop > 120 || loading) {
        return;
      }

      setLoading(true);
      const previousHeight = element.scrollHeight;
      try {
        let response: MessageListResponse;
        try {
          const remote = await api.get<MessageListResponse>(
            `/channels/${channelId}/messages`,
            { params: { cursor, limit: 30 } }
          );
          response = remote.data;
        } catch (error) {
          if (!isLikelyNetworkError(error)) {
            throw error;
          }

          response = await mockListMessages({
            channelId,
            cursor,
            limit: 30,
          });
        }
        const ordered = [...response.messages].reverse();
        dispatch(prependMessages({ channelId, messages: ordered }));
        dispatch(
          setPagination({
            channelId,
            cursor: response.next_cursor ?? null,
            hasMore: response.has_more,
          })
        );

        requestAnimationFrame(() => {
          const nextHeight = element.scrollHeight;
          element.scrollTop += nextHeight - previousHeight;
        });
      } finally {
        setLoading(false);
      }
    };

    element.addEventListener("scroll", onScroll);
    return () => element.removeEventListener("scroll", onScroll);
  }, [channelId, cursor, dispatch, hasMore, loading]);

  useLayoutEffect(() => {
    const element = parentRef.current;
    if (!element) {
      return;
    }

    const distanceFromBottom =
      element.scrollHeight - element.scrollTop - element.clientHeight;
    const shouldStickToBottom = distanceFromBottom < 180;
    setShowJumpToLatest(!shouldStickToBottom);

    if (shouldStickToBottom) {
      requestAnimationFrame(() => {
        element.scrollTop = element.scrollHeight;
      });
    }
  }, [messages.length, channelId]);

  if (!channelId) {
    return (
      <div className="message-list-empty">
        Pick a channel to start messaging.
      </div>
    );
  }

  return (
    <section className="message-list-shell">
      <div className="message-scroll" ref={parentRef}>
        <div className="message-stack">
          {rows.map((row) => (
            <MessageBubble
              key={row.key}
              message={row.message}
              isOwn={row.message.sender_id === currentUserId}
            />
          ))}
        </div>
      </div>

      <TypingIndicator users={typingUsers} />

      {showJumpToLatest ? (
        <button
          className="jump-latest"
          onClick={() => {
            if (parentRef.current) {
              parentRef.current.scrollTop = parentRef.current.scrollHeight;
              setShowJumpToLatest(false);
            }
          }}
          type="button"
        >
          Jump to latest
        </button>
      ) : null}
    </section>
  );
}

import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { ArrowDown, MessageSquare } from "lucide-react";

import { useAppDispatch, useAppSelector } from "@/app/store/hooks";
import {
  prependMessages,
  setLastServerEventId,
  setMessagesForChannel,
  setPagination,
} from "@/app/store/messageSlice";
import MessageBubble from "@/components/chat/MessageBubble";
import TypingIndicator from "@/components/chat/TypingIndicator";
import { Button } from "@/components/ui/button";
import { api } from "@/shared/lib/api";
import { mockListMessages, shouldUseMockFallback } from "@/shared/lib/mockApi";
import type { ChannelMember, Message } from "@/shared/types";

interface MessageListProps {
  channelId: string | null;
  members?: ChannelMember[];
}

interface MessageListResponse {
  messages: Message[];
  next_cursor?: string;
  has_more: boolean;
}

export default function MessageList({ channelId, members = [] }: MessageListProps) {
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

  const scrollRef = useRef<HTMLDivElement | null>(null);
  const [loading, setLoading] = useState(false);
  const [showJump, setShowJump] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const rows = useMemo(
    () =>
      messages.map((message, index) => ({
        key: message.id || `msg-${index}`,
        message,
      })),
    [messages]
  );
  const memberNameById = useMemo(
    () =>
      Object.fromEntries(
        members.map((member) => [
          member.user_id,
          member.display_name || member.username || member.user_id,
        ])
      ),
    [members]
  );

  useEffect(() => {
    if (!channelId) return;
    let cancelled = false;

    const loadInitial = async () => {
      setLoading(true);
      try {
        let response: MessageListResponse;
        try {
          const remote = await api.get<MessageListResponse>(`/channels/${channelId}/messages`, {
            params: { limit: 40 },
          });
          response = remote.data;
        } catch (error) {
          if (!shouldUseMockFallback(error)) {
            if (!cancelled) {
              setLoadError(error instanceof Error ? error.message : "Failed to load messages.");
            }
            return;
          }
          response = await mockListMessages({ channelId, limit: 40 });
        }
        if (cancelled) return;

        setLoadError(null);
        const ordered = [...response.messages].reverse();
        dispatch(setMessagesForChannel({ channelId, messages: ordered }));
        const latestEventId = [...ordered]
          .reverse()
          .find((message) => message.server_event_id)?.server_event_id;
        if (latestEventId) {
          dispatch(setLastServerEventId(latestEventId));
        }
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
    if (!channelId || !hasMore || !cursor || !scrollRef.current || loading) return;

    const element = scrollRef.current;
    const onScroll = async () => {
      if (element.scrollTop > 120 || loading) return;

      setLoading(true);
      const previousHeight = element.scrollHeight;
      try {
        let response: MessageListResponse;
        try {
          const remote = await api.get<MessageListResponse>(`/channels/${channelId}/messages`, {
            params: { cursor, limit: 30 },
          });
          response = remote.data;
        } catch (error) {
          if (!shouldUseMockFallback(error)) {
            setLoadError(error instanceof Error ? error.message : "Failed to load older messages.");
            return;
          }
          response = await mockListMessages({ channelId, cursor, limit: 30 });
        }

        setLoadError(null);
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
          element.scrollTop += element.scrollHeight - previousHeight;
        });
      } finally {
        setLoading(false);
      }
    };

    element.addEventListener("scroll", onScroll);
    return () => element.removeEventListener("scroll", onScroll);
  }, [channelId, cursor, dispatch, hasMore, loading]);

  useLayoutEffect(() => {
    const element = scrollRef.current;
    if (!element) return;

    const distanceFromBottom = element.scrollHeight - element.scrollTop - element.clientHeight;
    const stickToBottom = distanceFromBottom < 180;
    setShowJump(!stickToBottom);

    if (stickToBottom) {
      requestAnimationFrame(() => {
        element.scrollTop = element.scrollHeight;
      });
    }
  }, [messages.length, channelId]);

  if (!channelId) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-4 text-center text-muted-foreground">
        <div className="flex h-16 w-16 items-center justify-center rounded-[1.5rem] border border-border/70 bg-card/70">
          <MessageSquare className="h-7 w-7 opacity-60" />
        </div>
        <div className="space-y-1">
          <p className="text-base font-medium text-foreground">No conversation selected</p>
          <p className="text-sm text-muted-foreground">
            Open a channel from the sidebar to view messages and presence.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="relative flex min-h-0 flex-1 flex-col">
      <div ref={scrollRef} className="flex-1 overflow-y-auto overflow-x-hidden px-3 py-5 md:px-5">
        <div className="mx-auto flex w-full max-w-4xl flex-col gap-2">
          {loadError && (
            <p className="rounded-2xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive">
              {loadError}
            </p>
          )}
          {rows.map((row) => (
            <MessageBubble
              key={row.key}
              message={row.message}
              isOwn={row.message.sender_id === currentUserId}
              senderLabel={memberNameById[row.message.sender_id]}
            />
          ))}
        </div>
      </div>

      <TypingIndicator users={typingUsers} />

      {showJump && (
        <Button
          size="sm"
          onClick={() => {
            if (scrollRef.current) {
              scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
              setShowJump(false);
            }
          }}
          className="absolute bottom-16 right-6 gap-1.5"
        >
          <ArrowDown className="h-3.5 w-3.5" />
          Latest messages
        </Button>
      )}
    </div>
  );
}

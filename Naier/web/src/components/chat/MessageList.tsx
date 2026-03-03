import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { ArrowDown, MessageSquare } from "lucide-react";

import { useAppDispatch, useAppSelector } from "@/app/store/hooks";
import {
  setLastServerEventId,
  setPagination,
  setMessagesForChannel,
  prependMessages,
} from "@/app/store/messageSlice";
import { api } from "@/shared/lib/api";
import { isLikelyNetworkError, mockListMessages } from "@/shared/lib/mockApi";
import type { Message } from "@/shared/types";
import MessageBubble from "@/components/chat/MessageBubble";
import TypingIndicator from "@/components/chat/TypingIndicator";
import { Button } from "@/components/ui/button";

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

  const scrollRef = useRef<HTMLDivElement | null>(null);
  const [loading, setLoading] = useState(false);
  const [showJump, setShowJump] = useState(false);

  const rows = useMemo(
    () =>
      messages.map((message, index) => ({
        key: message.id || `msg-${index}`,
        message,
      })),
    [messages]
  );

  // 초기 메시지 로드
  useEffect(() => {
    if (!channelId) return;
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
          if (!isLikelyNetworkError(error)) throw error;
          response = await mockListMessages({ channelId, limit: 40 });
        }
        if (cancelled) return;

        const ordered = [...response.messages].reverse();
        dispatch(setMessagesForChannel({ channelId, messages: ordered }));
        const latestEventId = [...ordered]
          .reverse()
          .find((m) => m.server_event_id)?.server_event_id;
        if (latestEventId) dispatch(setLastServerEventId(latestEventId));
        dispatch(
          setPagination({
            channelId,
            cursor: response.next_cursor ?? null,
            hasMore: response.has_more,
          })
        );
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    void loadInitial();
    return () => { cancelled = true; };
  }, [channelId, dispatch]);

  // 무한 스크롤 — 위로 스크롤 시 이전 메시지 로드
  useEffect(() => {
    if (!channelId || !hasMore || !cursor || !scrollRef.current || loading) return;

    const element = scrollRef.current;
    const onScroll = async () => {
      if (element.scrollTop > 120 || loading) return;

      setLoading(true);
      const prevHeight = element.scrollHeight;
      try {
        let response: MessageListResponse;
        try {
          const remote = await api.get<MessageListResponse>(
            `/channels/${channelId}/messages`,
            { params: { cursor, limit: 30 } }
          );
          response = remote.data;
        } catch (error) {
          if (!isLikelyNetworkError(error)) throw error;
          response = await mockListMessages({ channelId, cursor, limit: 30 });
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
          element.scrollTop += element.scrollHeight - prevHeight;
        });
      } finally {
        setLoading(false);
      }
    };

    element.addEventListener("scroll", onScroll);
    return () => element.removeEventListener("scroll", onScroll);
  }, [channelId, cursor, dispatch, hasMore, loading]);

  // 새 메시지 도착 시 하단 고정
  useLayoutEffect(() => {
    const el = scrollRef.current;
    if (!el) return;

    const dist = el.scrollHeight - el.scrollTop - el.clientHeight;
    const stick = dist < 180;
    setShowJump(!stick);

    if (stick) {
      requestAnimationFrame(() => {
        el.scrollTop = el.scrollHeight;
      });
    }
  }, [messages.length, channelId]);

  if (!channelId) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-3 text-muted-foreground">
        <MessageSquare className="h-12 w-12 opacity-30" />
        <p className="text-sm">채널을 선택하면 대화가 시작됩니다</p>
      </div>
    );
  }

  return (
    <div className="relative flex flex-1 flex-col min-h-0">
      <div ref={scrollRef} className="flex-1 overflow-y-auto overflow-x-hidden px-2 py-4">
        <div className="flex flex-col gap-1.5">
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

      {showJump && (
        <Button
          size="sm"
          onClick={() => {
            if (scrollRef.current) {
              scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
              setShowJump(false);
            }
          }}
          className="absolute bottom-14 right-4 gap-1.5 shadow-lg"
        >
          <ArrowDown className="h-3.5 w-3.5" />
          최신 메시지
        </Button>
      )}
    </div>
  );
}

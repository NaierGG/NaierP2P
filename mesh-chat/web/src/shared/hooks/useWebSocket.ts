import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import type { AppDispatch } from "@/app/store";
import { useAppDispatch, useAppSelector } from "@/app/store/hooks";
import { incrementUnread, setLastMessage } from "@/app/store/channelSlice";
import {
  addMessage,
  applyReaction,
  markMessageDeleted,
  reconcilePendingMessage,
  updateMessage,
} from "@/app/store/messageSlice";
import { notifyIncomingMessage } from "@/shared/lib/browserNotifications";
import { mockHandleClientEvent } from "@/shared/lib/mockApi";
import { setPresence, setTyping } from "@/app/store/presenceSlice";
import { WSClient, type ConnectionState } from "@/shared/lib/websocket";
import type {
  ErrorPayload,
  MemberJoinedPayload,
  MemberLeftPayload,
  Message,
  MessageDeletedPayload,
  PresencePayload,
  ReactionPayload,
  TypingPayload,
  WSEvent,
} from "@/shared/types";

type EventHandler<TPayload = unknown> = (event: WSEvent<TPayload>) => void;

export function useWebSocket() {
  const dispatch = useAppDispatch();
  const accessToken = useAppSelector((state) => state.auth.accessToken);
  const currentUserId = useAppSelector((state) => state.auth.user?.id ?? null);
  const currentUser = useAppSelector((state) => state.auth.user);
  const activeChannelId = useAppSelector((state) => state.channels.activeChannelId);
  const clientRef = useRef<WSClient | null>(null);
  const handlersRef = useRef(new Map<string, Set<EventHandler>>());
  const activeChannelIdRef = useRef<string | null>(activeChannelId);
  const currentUserIdRef = useRef<string | null>(currentUserId);
  const [connectionState, setConnectionState] = useState<ConnectionState>("disconnected");

  const client = useMemo(() => new WSClient(() => accessToken), [accessToken]);

  useEffect(() => {
    activeChannelIdRef.current = activeChannelId;
  }, [activeChannelId]);

  useEffect(() => {
    currentUserIdRef.current = currentUserId;
  }, [currentUserId]);

  useEffect(() => {
    clientRef.current = client;

    const unsubscribers = [
      client.on("state", (state) => setConnectionState(state)),
      client.on("message", (event) => {
        routeEvent(event, {
          activeChannelId: activeChannelIdRef.current,
          currentUserId: currentUserIdRef.current,
          dispatch,
        });

        const handlers = handlersRef.current.get(event.type);
        handlers?.forEach((handler) => handler(event));
      }),
    ];

    client.connect();

    return () => {
      unsubscribers.forEach((unsubscribe) => unsubscribe());
      client.disconnect();
      clientRef.current = null;
    };
  }, [client, dispatch]);

  const send = useCallback((event: WSEvent) => {
    const clientInstance = clientRef.current;
    const state = clientInstance?.getState() ?? "disconnected";

    if (state === "connected") {
      clientInstance?.send(event);
      return;
    }

    void (async () => {
      const mockEvents = await mockHandleClientEvent({
        event,
        currentUser,
      });

      for (const mockEvent of mockEvents) {
        routeEvent(mockEvent, {
          activeChannelId: activeChannelIdRef.current,
          currentUserId: currentUserIdRef.current,
          dispatch,
        });

        const handlers = handlersRef.current.get(mockEvent.type);
        handlers?.forEach((handler) => handler(mockEvent));
      }
    })();
  }, [currentUser, dispatch]);

  const on = useCallback(function <TPayload = unknown>(
    eventType: string,
    handler: EventHandler<TPayload>
  ) {
    const handlers =
      handlersRef.current.get(eventType) ?? new Set<EventHandler>();
    handlers.add(handler as EventHandler);
    handlersRef.current.set(eventType, handlers);

    return () => {
      handlers.delete(handler as EventHandler);
      if (handlers.size === 0) {
        handlersRef.current.delete(eventType);
      }
    };
  }, []);

  const connect = useCallback(() => clientRef.current?.connect(), []);
  const disconnect = useCallback(() => clientRef.current?.disconnect(), []);

  return {
    connectionState,
    connect,
    disconnect,
    send,
    on,
  };
}

function routeEvent(
  event: WSEvent,
  context: {
    dispatch: AppDispatch;
    activeChannelId: string | null;
    currentUserId: string | null;
  }
) {
  switch (event.type) {
    case "MESSAGE_NEW": {
      const message = event.payload as Message;
      if (context.currentUserId && message.sender_id === context.currentUserId) {
        context.dispatch(reconcilePendingMessage(message));
      }
      context.dispatch(addMessage({ channelId: message.channel_id, message }));
      context.dispatch(setLastMessage({ channelId: message.channel_id, message }));
      if (
        context.activeChannelId !== message.channel_id &&
        message.sender_id !== context.currentUserId
      ) {
        context.dispatch(incrementUnread(message.channel_id));
        void notifyIncomingMessage(message);
      }
      break;
    }
    case "MESSAGE_UPDATED": {
      const message = event.payload as Message;
      context.dispatch(updateMessage({ channelId: message.channel_id, message }));
      context.dispatch(setLastMessage({ channelId: message.channel_id, message }));
      break;
    }
    case "MESSAGE_DELETED": {
      const payload = event.payload as MessageDeletedPayload;
      context.dispatch(
        markMessageDeleted({
          channelId: payload.channelId,
          messageId: payload.messageId,
        })
      );
      break;
    }
    case "TYPING": {
      const payload = event.payload as TypingPayload;
      context.dispatch(
        setTyping({
          channelId: payload.channelId,
          userId: payload.userId,
          isTyping: payload.isTyping,
        })
      );
      break;
    }
    case "REACTION": {
      const payload = event.payload as ReactionPayload;
      context.dispatch(
        applyReaction({
          messageId: payload.messageId,
          emoji: payload.emoji,
          userId: payload.userId,
          action: payload.action,
        })
      );
      break;
    }
    case "PRESENCE": {
      const payload = event.payload as PresencePayload;
      context.dispatch(
        setPresence({
          userId: payload.userId,
          status: payload.status,
        })
      );
      break;
    }
    case "MEMBER_JOINED": {
      const payload = event.payload as MemberJoinedPayload;
      void payload.user;
      break;
    }
    case "MEMBER_LEFT": {
      const payload = event.payload as MemberLeftPayload;
      void payload.userId;
      break;
    }
    case "ERROR": {
      const payload = event.payload as ErrorPayload;
      console.error("websocket error event", payload.code, payload.message);
      break;
    }
    default:
      break;
  }
}

export type { ConnectionState };

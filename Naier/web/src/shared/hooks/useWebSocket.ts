import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import type { AppDispatch } from "@/app/store";
import { useAppDispatch, useAppSelector } from "@/app/store/hooks";
import {
  incrementUnread,
  setLastMessage,
  setReadState,
} from "@/app/store/channelSlice";
import {
  addMessage,
  applyReaction,
  markMessageDeleted,
  reconcilePendingMessage,
  setLastServerEventId,
  updateMessage,
} from "@/app/store/messageSlice";
import { notifyIncomingMessage } from "@/shared/lib/browserNotifications";
import { api } from "@/shared/lib/api";
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
  ReadStatePayload,
  ReactionPayload,
  SyncEventsResponse,
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
  const lastServerEventId = useAppSelector((state) => state.messages.lastServerEventId);
  const clientRef = useRef<WSClient | null>(null);
  const handlersRef = useRef(new Map<string, Set<EventHandler>>());
  const activeChannelIdRef = useRef<string | null>(activeChannelId);
  const currentUserIdRef = useRef<string | null>(currentUserId);
  const lastServerEventIdRef = useRef<string | null>(lastServerEventId);
  const [connectionState, setConnectionState] = useState<ConnectionState>("disconnected");

  const client = useMemo(() => new WSClient(() => accessToken), [accessToken]);

  useEffect(() => {
    activeChannelIdRef.current = activeChannelId;
  }, [activeChannelId]);

  useEffect(() => {
    currentUserIdRef.current = currentUserId;
  }, [currentUserId]);

  useEffect(() => {
    lastServerEventIdRef.current = lastServerEventId;
  }, [lastServerEventId]);

  useEffect(() => {
    clientRef.current = client;

    const unsubscribers = [
      client.on("state", (state) => setConnectionState(state)),
      client.on("message", (event) => {
        if (event.type === "MESSAGE_NEW") {
          const payload = event.payload as Message;
          if (payload.id) {
            client.send({
              type: "DELIVERY_ACK",
              payload: {
                messageId: payload.id,
              },
            });
          }
        }

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

  useEffect(() => {
    if (connectionState !== "connected" || !accessToken) {
      return;
    }

    let cancelled = false;
    void (async () => {
      try {
        const response = await api.get<SyncEventsResponse>("/events/sync", {
          params: {
            after: lastServerEventIdRef.current ?? undefined,
            limit: 200,
          },
        });

        if (cancelled) {
          return;
        }

        for (const event of response.data.events) {
          const payload = event.message ?? event.reaction ?? event.read_state;
          if (!payload) {
            continue;
          }
          if (event.type === "MESSAGE_NEW" && event.message?.id) {
            client.send({
              type: "DELIVERY_ACK",
              payload: {
                messageId: event.message.id,
              },
            });
          }
          routeEvent(
            {
              type: event.type,
              payload,
            },
            {
              activeChannelId: activeChannelIdRef.current,
              currentUserId: currentUserIdRef.current,
              dispatch,
            }
          );
        }

        if (response.data.last_event_id) {
          dispatch(setLastServerEventId(response.data.last_event_id));
        }
      } catch (error) {
        console.error("sync catch-up failed", error);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [accessToken, connectionState, dispatch]);

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
      if (message.server_event_id) {
        context.dispatch(setLastServerEventId(message.server_event_id));
      }
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
      if (message.server_event_id) {
        context.dispatch(setLastServerEventId(message.server_event_id));
      }
      context.dispatch(updateMessage({ channelId: message.channel_id, message }));
      context.dispatch(setLastMessage({ channelId: message.channel_id, message }));
      break;
    }
    case "MESSAGE_DELETED": {
      const payload = event.payload as Message | MessageDeletedPayload;
      if ("server_event_id" in payload && payload.server_event_id) {
        context.dispatch(setLastServerEventId(payload.server_event_id));
        context.dispatch(
          markMessageDeleted({
            channelId: payload.channel_id,
            messageId: payload.id,
          })
        );
      } else {
        context.dispatch(
          markMessageDeleted({
            channelId: (payload as MessageDeletedPayload).channelId,
            messageId: (payload as MessageDeletedPayload).messageId,
          })
        );
      }
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
      if (payload.event_id) {
        context.dispatch(setLastServerEventId(payload.event_id));
      }
      context.dispatch(
        applyReaction({
          messageId: payload.message_id ?? payload.messageId ?? "",
          emoji: payload.emoji,
          userId: payload.user_id ?? payload.userId ?? "",
          action: payload.action,
        })
      );
      break;
    }
    case "READ_STATE": {
      const payload = event.payload as ReadStatePayload;
      if (payload.event_id) {
        context.dispatch(setLastServerEventId(payload.event_id));
      }
      const channelId = payload.channel_id ?? payload.channelId;
      const userId = payload.user_id ?? payload.userId;
      const lastReadSequence =
        payload.last_read_sequence ?? payload.lastReadSequence ?? 0;
      if (channelId && userId) {
        context.dispatch(
          setReadState({
            channelId,
            userId,
            lastReadSequence,
          })
        );
      }
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

import { useEffect, useMemo, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { LogOut, Settings, Search } from "lucide-react";

import { clearAuth } from "@/app/store/authSlice";
import { useAppDispatch, useAppSelector } from "@/app/store/hooks";
import { setActiveChannel, setChannels } from "@/app/store/channelSlice";

import AppShell from "@/components/layout/AppShell";
import SidebarLayout from "@/components/layout/Sidebar";
import ChatPanel from "@/components/layout/ChatPanel";
import ChatHeader from "@/components/chat/ChatHeader";
import ChannelItem from "@/components/chat/ChannelItem";
import MessageList from "@/components/chat/MessageList";
import Composer from "@/components/chat/Composer";

import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { ScrollArea } from "@/components/ui/scroll-area";

import { api } from "@/shared/lib/api";
import { useWebSocket } from "@/shared/hooks/useWebSocket";
import { consumePendingNotificationChannel } from "@/shared/lib/browserNotifications";
import { isLikelyNetworkError, mockListChannels } from "@/shared/lib/mockApi";
import { useSettings } from "@/features/settings/useSettings";
import type { Channel } from "@/shared/types";

interface ChannelListResponse {
  channels: Channel[];
}

export default function ChannelList() {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const { connectionState, send } = useWebSocket();
  const { settings } = useSettings();
  const currentUser = useAppSelector((state) => state.auth.user);
  const accessToken = useAppSelector((state) => state.auth.accessToken);
  const channels = useAppSelector((state) => state.channels.channels);
  const activeChannelId = useAppSelector((state) => state.channels.activeChannelId);
  const activeChannel = useMemo(
    () => (activeChannelId ? channels[activeChannelId] ?? null : null),
    [activeChannelId, channels]
  );
  const joinedChannelsRef = useRef<Set<string>>(new Set());
  const pendingNotificationChannelRef = useRef<string | null>(consumePendingNotificationChannel());

  const channelList = useMemo(
    () =>
      Object.values(channels).sort((left, right) => {
        const leftTime = left.last_message?.created_at ?? left.created_at;
        const rightTime = right.last_message?.created_at ?? right.created_at;
        return rightTime.localeCompare(leftTime);
      }),
    [channels]
  );
  const isMockMode = accessToken?.startsWith("mock-access-") ?? false;

  useEffect(() => {
    let cancelled = false;

    const loadChannels = async () => {
      try {
        const response = await api.get<ChannelListResponse>("/channels");
        if (cancelled) return;

        dispatch(setChannels(response.data.channels));
        const pendingChannelId = pendingNotificationChannelRef.current;
        if (
          pendingChannelId &&
          response.data.channels.some((channel) => channel.id === pendingChannelId)
        ) {
          dispatch(setActiveChannel(pendingChannelId));
          pendingNotificationChannelRef.current = null;
        } else if (!activeChannelId && response.data.channels.length > 0) {
          dispatch(setActiveChannel(response.data.channels[0].id));
        }
      } catch (error) {
        if (!isLikelyNetworkError(error)) throw error;

        const response = await mockListChannels();
        if (cancelled) return;

        dispatch(setChannels(response.channels));
        const pendingChannelId = pendingNotificationChannelRef.current;
        if (
          pendingChannelId &&
          response.channels.some((channel) => channel.id === pendingChannelId)
        ) {
          dispatch(setActiveChannel(pendingChannelId));
          pendingNotificationChannelRef.current = null;
        } else if (!activeChannelId && response.channels.length > 0) {
          dispatch(setActiveChannel(response.channels[0].id));
        }
      }
    };

    void loadChannels();
    return () => { cancelled = true; };
  }, [dispatch]);

  useEffect(() => {
    const nextChannelIds = new Set(channelList.map((ch) => ch.id));

    for (const channelId of nextChannelIds) {
      if (!joinedChannelsRef.current.has(channelId)) {
        send({ type: "CHANNEL_JOIN", payload: { channelId } });
        joinedChannelsRef.current.add(channelId);
      }
    }

    for (const channelId of joinedChannelsRef.current) {
      if (!nextChannelIds.has(channelId)) {
        send({ type: "CHANNEL_LEAVE", payload: { channelId } });
        joinedChannelsRef.current.delete(channelId);
      }
    }
  }, [channelList, send]);

  useEffect(() => {
    return () => {
      for (const channelId of joinedChannelsRef.current) {
        send({ type: "CHANNEL_LEAVE", payload: { channelId } });
      }
      joinedChannelsRef.current.clear();
    };
  }, [send]);

  return (
    <AppShell
      sidebar={
        <SidebarLayout>
          <div className="px-4 pt-5 pb-2">
            <h2 className="text-lg font-semibold tracking-tight">Naier</h2>
            <p className="text-xs text-muted-foreground">
              {currentUser?.display_name ?? currentUser?.username ?? "Guest"}
            </p>
          </div>

          <div className="px-3 pb-2">
            <div className="flex items-center gap-2 rounded-xl border border-input bg-card px-3 py-2 text-sm text-muted-foreground">
              <Search className="h-3.5 w-3.5" />
              <span>채널 검색</span>
            </div>
          </div>

          <ScrollArea className="flex-1">
            <div className="flex flex-col gap-0.5 px-2 py-1">
              {channelList.length === 0 ? (
                <p className="px-3 py-8 text-center text-sm text-muted-foreground">
                  채널이 없습니다
                </p>
              ) : (
                channelList.map((channel) => (
                  <ChannelItem
                    key={channel.id}
                    channel={channel}
                    isActive={channel.id === activeChannelId}
                    showPreview={settings.messagePreview}
                    onClick={() => dispatch(setActiveChannel(channel.id))}
                  />
                ))
              )}
            </div>
          </ScrollArea>

          <Separator />
          <div className="flex items-center justify-between gap-2 px-3 py-3">
            <div className="min-w-0">
              <p className="truncate text-sm font-medium">
                {currentUser?.display_name ?? currentUser?.username ?? "Guest"}
              </p>
              <p className="truncate text-xs text-muted-foreground">
                {currentUser?.username ? `@${currentUser.username}` : ""}
              </p>
            </div>
            <div className="flex gap-1">
              <Button
                variant="ghost"
                size="icon"
                onClick={() => navigate("/app/settings")}
                title="설정"
              >
                <Settings className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => {
                  dispatch(clearAuth());
                  navigate("/auth/login");
                }}
                title="로그아웃"
              >
                <LogOut className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </SidebarLayout>
      }
    >
      <ChatPanel>
        <ChatHeader
          channel={activeChannel}
          connectionState={connectionState}
          isMockMode={isMockMode}
        />
        <MessageList channelId={activeChannelId} />
        <Composer channelId={activeChannelId} send={send} />
      </ChatPanel>
    </AppShell>
  );
}

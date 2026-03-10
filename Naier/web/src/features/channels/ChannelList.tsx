import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { LogOut, Search, Settings } from "lucide-react";

import { clearAuth } from "@/app/store/authSlice";
import { useAppDispatch, useAppSelector } from "@/app/store/hooks";
import { setActiveChannel, setChannels } from "@/app/store/channelSlice";
import ChannelItem from "@/components/chat/ChannelItem";
import ChatHeader from "@/components/chat/ChatHeader";
import Composer from "@/components/chat/Composer";
import MemberBar from "@/components/chat/MemberBar";
import MessageList from "@/components/chat/MessageList";
import ChatPanel from "@/components/layout/ChatPanel";
import AppShell from "@/components/layout/AppShell";
import SidebarLayout from "@/components/layout/Sidebar";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { getChannelPresentation } from "@/features/channels/channelPresentation";
import { useSettings } from "@/features/settings/useSettings";
import { useWebSocket } from "@/shared/hooks/useWebSocket";
import { api } from "@/shared/lib/api";
import { consumePendingNotificationChannel } from "@/shared/lib/browserNotifications";
import {
  mockListChannelMembers,
  mockListChannels,
  shouldUseMockFallback,
} from "@/shared/lib/mockApi";
import type { Channel, ChannelMember } from "@/shared/types";

interface ChannelListResponse {
  channels: Channel[];
}

interface ChannelMembersResponse {
  members: ChannelMember[];
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

  const [channelMembers, setChannelMembers] = useState<Record<string, ChannelMember[]>>({});
  const [loadError, setLoadError] = useState<string | null>(null);
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

  const activePresentation = useMemo(
    () =>
      activeChannel
        ? getChannelPresentation(
            activeChannel,
            channelMembers[activeChannel.id],
            currentUser?.id ?? null,
            settings.messagePreview
          )
        : null,
    [activeChannel, channelMembers, currentUser?.id, settings.messagePreview]
  );

  const isMockMode = accessToken?.startsWith("mock-access-") ?? false;

  useEffect(() => {
    let cancelled = false;

    const loadChannels = async () => {
      try {
        const response = await api.get<ChannelListResponse>("/channels");
        if (cancelled) return;

        setLoadError(null);
        dispatch(setChannels(response.data.channels));
        selectInitialChannel(response.data.channels);
      } catch (error) {
        if (!shouldUseMockFallback(error)) {
          if (!cancelled) {
            setLoadError(error instanceof Error ? error.message : "Failed to load channels.");
          }
          return;
        }

        const response = await mockListChannels();
        if (cancelled) return;

        setLoadError(null);
        dispatch(setChannels(response.channels));
        selectInitialChannel(response.channels);
      }
    };

    const selectInitialChannel = (nextChannels: Channel[]) => {
      const pendingChannelId = pendingNotificationChannelRef.current;
      if (pendingChannelId && nextChannels.some((channel) => channel.id === pendingChannelId)) {
        dispatch(setActiveChannel(pendingChannelId));
        pendingNotificationChannelRef.current = null;
        return;
      }

      if (!activeChannelId && nextChannels.length > 0) {
        dispatch(setActiveChannel(nextChannels[0].id));
      }
    };

    void loadChannels();
    return () => {
      cancelled = true;
    };
  }, [activeChannelId, dispatch]);

  useEffect(() => {
    let cancelled = false;

    const loadDmMembers = async () => {
      const neededChannels = channelList.filter(
        (channel) =>
          !channelMembers[channel.id] &&
          (channel.type === "dm" || channel.id === activeChannelId)
      );
      if (neededChannels.length === 0) {
        return;
      }

      const responses = await Promise.all(
        neededChannels.map(async (channel) => {
          try {
            const response = await api.get<ChannelMembersResponse>(`/channels/${channel.id}/members`);
            return { channelId: channel.id, members: response.data.members };
          } catch (error) {
            if (!shouldUseMockFallback(error)) {
              return { channelId: channel.id, members: [] };
            }
            const mockResponse = await mockListChannelMembers(channel.id);
            return { channelId: channel.id, members: mockResponse.members };
          }
        })
      );

      if (cancelled) return;

      setChannelMembers((previous) => {
        const next = { ...previous };
        let changed = false;
        for (const response of responses) {
          next[response.channelId] = response.members;
          changed = true;
        }
        return changed ? next : previous;
      });
    };

    void loadDmMembers();
    return () => {
      cancelled = true;
    };
  }, [activeChannelId, channelList, channelMembers]);

  useEffect(() => {
    const nextChannelIds = new Set(channelList.map((channel) => channel.id));

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
          <div className="border-b border-border/60 px-5 pb-4 pt-6">
            <p className="mb-2 text-[11px] font-semibold uppercase tracking-[0.28em] text-primary/75">
              Trusted network
            </p>
            <h2 className="text-2xl font-semibold tracking-tight">Naier</h2>
            <p className="mt-1 text-sm text-muted-foreground">
              {currentUser?.display_name ?? currentUser?.username ?? "Guest"}
            </p>
          </div>

          <div className="px-4 pb-3 pt-4">
            <div className="flex items-center gap-2 rounded-[1.35rem] border border-input/80 bg-card/70 px-3.5 py-3 text-sm text-muted-foreground shadow-[inset_0_1px_0_rgba(255,255,255,0.02)]">
              <Search className="h-3.5 w-3.5" />
              <span>Search secure threads</span>
            </div>
          </div>

          {loadError && (
            <div className="px-4 pb-3">
              <p className="rounded-2xl border border-destructive/30 bg-destructive/10 px-3 py-3 text-xs text-destructive">
                {loadError}
              </p>
            </div>
          )}

          <ScrollArea className="flex-1">
            <div className="flex flex-col gap-2 px-3 py-2">
              {channelList.length === 0 ? (
                <p className="px-3 py-10 text-center text-sm text-muted-foreground">
                  No channels yet
                </p>
              ) : (
                channelList.map((channel) => {
                  const presentation = getChannelPresentation(
                    channel,
                    channelMembers[channel.id],
                    currentUser?.id ?? null,
                    settings.messagePreview
                  );

                  return (
                    <ChannelItem
                      key={channel.id}
                      channel={channel}
                      title={presentation.title}
                      preview={presentation.preview}
                      isActive={channel.id === activeChannelId}
                      onClick={() => dispatch(setActiveChannel(channel.id))}
                    />
                  );
                })
              )}
            </div>
          </ScrollArea>

          <Separator />
          <div className="flex items-center justify-between gap-3 px-4 py-4">
            <div className="min-w-0">
              <p className="truncate text-sm font-medium">
                {currentUser?.display_name ?? currentUser?.username ?? "Guest"}
              </p>
              <p className="truncate text-xs text-muted-foreground">
                {currentUser?.username ? `@${currentUser.username}` : ""}
              </p>
            </div>
            <div className="flex gap-2">
              <Button
                data-testid="open-settings"
                variant="outline"
                size="icon"
                onClick={() => navigate("/app/settings")}
                title="Settings"
              >
                <Settings className="h-4 w-4" />
              </Button>
              <Button
                variant="outline"
                size="icon"
                onClick={() => {
                  dispatch(clearAuth());
                  navigate("/auth/login");
                }}
                title="Log out"
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
          title={activePresentation?.title}
          meta={activePresentation?.meta}
          connectionState={connectionState}
          isMockMode={isMockMode}
        />
        <MemberBar members={activeChannelId ? channelMembers[activeChannelId] ?? [] : []} />
        <MessageList
          channelId={activeChannelId}
          members={activeChannelId ? channelMembers[activeChannelId] ?? [] : []}
        />
        <Composer channelId={activeChannelId} send={send} />
      </ChatPanel>
    </AppShell>
  );
}

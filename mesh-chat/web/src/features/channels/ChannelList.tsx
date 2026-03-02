import { useEffect, useMemo, useRef } from "react";
import { useNavigate } from "react-router-dom";

import { clearAuth } from "@/app/store/authSlice";
import { useAppDispatch, useAppSelector } from "@/app/store/hooks";
import { setActiveChannel, setChannels } from "@/app/store/channelSlice";
import MessageInput from "@/features/messages/MessageInput";
import MessageList from "@/features/messages/MessageList";
import { api } from "@/shared/lib/api";
import { useWebSocket } from "@/shared/hooks/useWebSocket";
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
        if (cancelled) {
          return;
        }

        dispatch(setChannels(response.data.channels));
        if (!activeChannelId && response.data.channels.length > 0) {
          dispatch(setActiveChannel(response.data.channels[0].id));
        }
      } catch (error) {
        if (!isLikelyNetworkError(error)) {
          throw error;
        }

        const response = await mockListChannels();
        if (cancelled) {
          return;
        }

        dispatch(setChannels(response.channels));
        if (!activeChannelId && response.channels.length > 0) {
          dispatch(setActiveChannel(response.channels[0].id));
        }
      }
    };

    void loadChannels();

    return () => {
      cancelled = true;
    };
  }, [dispatch]);

  useEffect(() => {
    const nextChannelIds = new Set(channelList.map((channel) => channel.id));

    for (const channelId of nextChannelIds) {
      if (!joinedChannelsRef.current.has(channelId)) {
        send({
          type: "CHANNEL_JOIN",
          payload: { channelId },
        });
        joinedChannelsRef.current.add(channelId);
      }
    }

    for (const channelId of joinedChannelsRef.current) {
      if (!nextChannelIds.has(channelId)) {
        send({
          type: "CHANNEL_LEAVE",
          payload: { channelId },
        });
        joinedChannelsRef.current.delete(channelId);
      }
    }
  }, [channelList, send]);

  useEffect(() => {
    return () => {
      for (const channelId of joinedChannelsRef.current) {
        send({
          type: "CHANNEL_LEAVE",
          payload: { channelId },
        });
      }
      joinedChannelsRef.current.clear();
    };
  }, [send]);

  return (
    <section className={`app-shell ${settings.compactSidebar ? "is-compact-sidebar" : ""}`}>
      <aside className="sidebar">
        <div className="sidebar-header">
          <p className="eyebrow">Inbox</p>
          <h2>Channels</h2>
          <p className="muted">
            Realtime: {isMockMode ? "local mock mode" : connectionState}
          </p>
          <p className="muted">
            Status: {settings.status === "dnd" ? "Do not disturb" : settings.status}
          </p>
          {isMockMode ? <div className="mock-pill">Backend offline fallback</div> : null}
        </div>
        <div className="channel-list">
          {channelList.length === 0 ? (
            <div className="empty-card">No channels loaded yet.</div>
          ) : (
            channelList.map((channel) => {
              const isActive = channel.id === activeChannelId;
              return (
                <button
                  className={`channel-card ${isActive ? "is-active" : ""}`}
                  key={channel.id}
                  onClick={() => dispatch(setActiveChannel(channel.id))}
                  type="button"
                >
                  <div className="channel-card-top">
                    <strong>{channel.name}</strong>
                    <span className="muted">
                      {new Date(
                        channel.last_message?.created_at ?? channel.created_at
                      ).toLocaleDateString()}
                    </span>
                  </div>
                  <p className="muted">
                    {settings.messagePreview
                      ? channel.last_message?.content || channel.description || channel.type
                      : channel.description || channel.type}
                  </p>
                </button>
              );
            })
          )}
        </div>

        <div className="sidebar-footer">
          <div className="sidebar-profile">
            <div>
              <strong>{currentUser?.display_name ?? currentUser?.username ?? "Guest"}</strong>
              <p className="muted">{currentUser?.username ? `@${currentUser.username}` : ""}</p>
            </div>
          </div>
          <button
            className="secondary-button"
            onClick={() => navigate("/app/settings")}
            type="button"
          >
            Open settings
          </button>
          <button
            className="secondary-button is-danger"
            onClick={() => {
              dispatch(clearAuth());
              navigate("/auth/login");
            }}
            type="button"
          >
            Sign out
          </button>
        </div>
      </aside>

      <main className="chat-stage chat-workspace">
        <header className="chat-header">
          <div>
            <p className="eyebrow">Conversation</p>
            <h1>{activeChannel?.name ?? "Select a channel"}</h1>
          </div>
          <p className="muted">
            {activeChannel
              ? `${activeChannel.member_count} members - ${activeChannel.type}`
              : "Choose a room from the sidebar"}
          </p>
        </header>

        <MessageList channelId={activeChannelId} />
        <MessageInput channelId={activeChannelId} send={send} />
      </main>
    </section>
  );
}

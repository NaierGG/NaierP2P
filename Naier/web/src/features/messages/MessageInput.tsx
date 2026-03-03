import { useEffect, useRef, useState } from "react";

import { useAppDispatch, useAppSelector } from "@/app/store/hooks";
import { addPendingMessage, failPendingMessage } from "@/app/store/messageSlice";
import { useSettings } from "@/features/settings/useSettings";
import { useEncryption } from "@/shared/hooks/useEncryption";
import type { WSEvent } from "@/shared/types";

interface MessageInputProps {
  channelId: string | null;
  send: (event: WSEvent) => void;
}

export default function MessageInput({ channelId, send }: MessageInputProps) {
  const dispatch = useAppDispatch();
  const currentUser = useAppSelector((state) => state.auth.user);
  const { settings } = useSettings();
  const [value, setValue] = useState("");
  const typingTimeoutRef = useRef<number | null>(null);
  const { loadChannelKey, encryptForChannel } = useEncryption();

  useEffect(() => {
    return () => {
      if (typingTimeoutRef.current) {
        window.clearTimeout(typingTimeoutRef.current);
      }
    };
  }, []);

  async function handleSend() {
    const trimmed = value.trim();
    if (!channelId || !currentUser || !trimmed) {
      return;
    }

    const clientId = crypto.randomUUID();
    const pendingMessage = {
      client_id: clientId,
      id: clientId,
      channel_id: channelId,
      sender_id: currentUser.id,
      type: "text" as const,
      content: trimmed,
      is_edited: false,
      is_deleted: false,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      status: "sending" as const,
      reactions: [],
    };

    dispatch(addPendingMessage(pendingMessage));

    try {
      let content = trimmed;
      let iv = "";

      const existingChannelKey = await loadChannelKey(channelId);
      if (existingChannelKey) {
        const encrypted = await encryptForChannel(channelId, trimmed);
        content = encrypted.ciphertext;
        iv = encrypted.iv;
      }

      send({
        type: "MESSAGE_SEND",
        request_id: clientId,
        payload: {
          channelId,
          content,
          iv,
          clientEventId: clientId,
        },
      });

      setValue("");
      emitTyping(false);
    } catch (error) {
      dispatch(
        failPendingMessage({
          clientId,
          error: error instanceof Error ? error.message : "Failed to send message",
        })
      );
    }
  }

  function emitTyping(isTyping: boolean) {
    if (!channelId) {
      return;
    }

    send({
      type: isTyping ? "TYPING_START" : "TYPING_STOP",
      payload: {
        channelId,
      },
    });
  }

  function handleChange(nextValue: string) {
    setValue(nextValue);

    if (!channelId) {
      return;
    }

    emitTyping(nextValue.trim().length > 0);

    if (typingTimeoutRef.current) {
      window.clearTimeout(typingTimeoutRef.current);
    }

    typingTimeoutRef.current = window.setTimeout(() => {
      emitTyping(false);
    }, 2000);
  }

  return (
    <div className="message-input-shell">
      <textarea
        className="message-input"
        disabled={!channelId}
        onChange={(event) => handleChange(event.target.value)}
        onKeyDown={(event) => {
          if (settings.enterToSend && event.key === "Enter" && !event.shiftKey) {
            event.preventDefault();
            void handleSend();
          }
        }}
        placeholder={
          channelId
            ? settings.enterToSend
              ? "Write a message. Enter sends, Shift+Enter adds a line."
              : "Write a message. Use the Send button to submit."
            : "Select a channel first"
        }
        rows={1}
        value={value}
      />
      <button
        className="primary-button"
        disabled={!channelId || value.trim().length === 0}
        onClick={() => void handleSend()}
        type="button"
      >
        Send
      </button>
    </div>
  );
}

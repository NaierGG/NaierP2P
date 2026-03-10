import { useEffect, useRef, useState } from "react";
import { Send } from "lucide-react";

import { useAppDispatch, useAppSelector } from "@/app/store/hooks";
import { addPendingMessage, failPendingMessage } from "@/app/store/messageSlice";
import { Button } from "@/components/ui/button";
import { useSettings } from "@/features/settings/useSettings";
import { useEncryption } from "@/shared/hooks/useEncryption";
import type { WSEvent } from "@/shared/types";

interface ComposerProps {
  channelId: string | null;
  send: (event: WSEvent) => void;
}

export default function Composer({ channelId, send }: ComposerProps) {
  const dispatch = useAppDispatch();
  const currentUser = useAppSelector((state) => state.auth.user);
  const { settings } = useSettings();
  const [value, setValue] = useState("");
  const typingTimeoutRef = useRef<number | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const { loadChannelKey, encryptForChannel } = useEncryption();

  useEffect(() => {
    return () => {
      if (typingTimeoutRef.current) {
        window.clearTimeout(typingTimeoutRef.current);
      }
    };
  }, []);

  // Keep the input compact while still allowing short multi-line drafts.
  useEffect(() => {
    const element = textareaRef.current;
    if (!element) return;
    element.style.height = "0";
    element.style.height = `${Math.min(element.scrollHeight, 160)}px`;
  }, [value]);

  async function handleSend() {
    const trimmed = value.trim();
    if (!channelId || !currentUser || !trimmed) return;

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
        payload: { channelId, content, iv, clientEventId: clientId },
      });

      setValue("");
      emitTyping(false);
    } catch (error) {
      dispatch(
        failPendingMessage({
          clientId,
          error: error instanceof Error ? error.message : "Message delivery failed.",
        })
      );
    }
  }

  function emitTyping(isTyping: boolean) {
    if (!channelId) return;
    send({
      type: isTyping ? "TYPING_START" : "TYPING_STOP",
      payload: { channelId },
    });
  }

  function handleChange(nextValue: string) {
    setValue(nextValue);
    if (!channelId) return;
    emitTyping(nextValue.trim().length > 0);
    if (typingTimeoutRef.current) window.clearTimeout(typingTimeoutRef.current);
    typingTimeoutRef.current = window.setTimeout(() => emitTyping(false), 2000);
  }

  const disabled = !channelId;

  return (
    <div className="border-t border-border/70 bg-card/45 px-4 py-4 backdrop-blur-sm md:px-6">
      <div className="mx-auto flex w-full max-w-4xl items-end gap-3">
        <textarea
          data-testid="chat-composer-input"
          ref={textareaRef}
          className="flex-1 resize-none rounded-[1.5rem] border border-input/80 bg-card/70 px-4 py-3.5 text-sm text-foreground shadow-[inset_0_1px_0_rgba(255,255,255,0.02)] placeholder:text-muted-foreground/85 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
          disabled={disabled}
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
                ? "Write a secure message. Press Enter to send."
                : "Write a secure message. Shift+Enter adds a line."
              : "Select a channel to start composing."
          }
          rows={1}
          value={value}
        />
        <Button
          data-testid="chat-composer-send"
          size="icon"
          disabled={disabled || value.trim().length === 0}
          onClick={() => void handleSend()}
          className="h-12 w-12 shrink-0"
        >
          <Send className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}

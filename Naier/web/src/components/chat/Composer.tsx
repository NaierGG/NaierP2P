import { useEffect, useRef, useState } from "react";
import { Send } from "lucide-react";

import { useAppDispatch, useAppSelector } from "@/app/store/hooks";
import { addPendingMessage, failPendingMessage } from "@/app/store/messageSlice";
import { useSettings } from "@/features/settings/useSettings";
import { useEncryption } from "@/shared/hooks/useEncryption";
import { Button } from "@/components/ui/button";
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

  // 텍스트영역 높이 자동 조절 — 사용자가 입력 크기를 볼 수 있어 인지 부하 감소
  useEffect(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = "0";
    el.style.height = `${Math.min(el.scrollHeight, 160)}px`;
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
          error: error instanceof Error ? error.message : "전송 실패",
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
    <div className="border-t border-border px-4 py-3">
      <div className="flex items-end gap-2">
        <textarea
          ref={textareaRef}
          className="flex-1 resize-none rounded-xl border border-input bg-card px-4 py-3 text-sm text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
          disabled={disabled}
          onChange={(e) => handleChange(e.target.value)}
          onKeyDown={(e) => {
            if (settings.enterToSend && e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              void handleSend();
            }
          }}
          placeholder={
            channelId
              ? settings.enterToSend
                ? "메시지를 입력하세요 (Enter로 전송)"
                : "메시지를 입력하세요"
              : "채널을 선택하세요"
          }
          rows={1}
          value={value}
        />
        <Button
          size="icon"
          disabled={disabled || value.trim().length === 0}
          onClick={() => void handleSend()}
          className="h-10 w-10 shrink-0"
        >
          <Send className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}

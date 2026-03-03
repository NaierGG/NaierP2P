import { useSettings } from "@/features/settings/useSettings";

interface TypingIndicatorProps {
  users: string[];
}

export default function TypingIndicator({ users }: TypingIndicatorProps) {
  const { settings } = useSettings();

  if (!settings.typingIndicators || users.length === 0) return null;

  const label =
    users.length === 1
      ? `${users[0].slice(0, 8)} 입력 중...`
      : `${users.slice(0, 2).map((u) => u.slice(0, 8)).join(", ")} 입력 중...`;

  return (
    <div className="flex items-center gap-2 px-6 py-2 text-xs text-muted-foreground">
      <span className="inline-flex gap-1">
        <i className="typing-dot" />
        <i className="typing-dot" />
        <i className="typing-dot" />
      </span>
      <span>{label}</span>
    </div>
  );
}

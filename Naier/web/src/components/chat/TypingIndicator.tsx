import { useSettings } from "@/features/settings/useSettings";

interface TypingIndicatorProps {
  users: string[];
}

export default function TypingIndicator({ users }: TypingIndicatorProps) {
  const { settings } = useSettings();

  if (!settings.typingIndicators || users.length === 0) return null;

  const label =
    users.length === 1
      ? `${users[0].slice(0, 8)} is typing...`
      : `${users.slice(0, 2).map((user) => user.slice(0, 8)).join(", ")} are typing...`;

  return (
    <div className="mx-auto flex w-full max-w-4xl items-center gap-2 px-6 py-2 text-xs text-muted-foreground">
      <span className="inline-flex gap-1">
        <i className="typing-dot" />
        <i className="typing-dot" />
        <i className="typing-dot" />
      </span>
      <span>{label}</span>
    </div>
  );
}

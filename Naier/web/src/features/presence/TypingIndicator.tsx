import { useSettings } from "@/features/settings/useSettings";

interface TypingIndicatorProps {
  users: string[];
}

export default function TypingIndicator({ users }: TypingIndicatorProps) {
  const { settings } = useSettings();

  if (!settings.typingIndicators || users.length === 0) {
    return null;
  }

  const label =
    users.length === 1
      ? `${users[0].slice(0, 8)} is typing...`
      : `${users.slice(0, 2).map((user) => user.slice(0, 8)).join(", ")} are typing...`;

  return (
    <div className="typing-indicator">
      <span>{label}</span>
      <span className="typing-dots">
        <i />
        <i />
        <i />
      </span>
    </div>
  );
}

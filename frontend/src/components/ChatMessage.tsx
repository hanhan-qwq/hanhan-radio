import type { Message } from "../types";

interface Props {
  message: Message;
}

export default function ChatMessage({ message }: Props) {
  const isUser = message.role === "user";
  const isLoading =
    !isUser &&
    message.episodeStatus &&
    message.episodeStatus !== "done" &&
    message.episodeStatus !== "failed";

  const isFailed =
    !isUser && message.episodeStatus === "failed";

  return (
    <div className={`flex ${isUser ? "justify-end" : "justify-start"}`}>
      <div
        className={`max-w-[80%] rounded-xl px-4 py-2.5 text-sm leading-relaxed ${
          isUser
            ? "bg-[var(--color-accent)] text-white rounded-br-md"
            : "bg-[var(--color-bubble-assistant)] text-[var(--color-text-primary)] rounded-bl-md border border-[var(--color-border)]"
        }`}
      >
        {isLoading ? (
          <div className="flex items-center gap-2">
            <span className="animate-pulse-soft">🎵</span>
            <span className="text-[var(--color-text-secondary)]">
              {message.episodeStatus === "pending" ? "正在准备..." : "AI 正在为您生成电台节目..."}
            </span>
          </div>
        ) : isFailed ? (
          <p className="text-red-400">{message.text || "生成失败，请重试"}</p>
        ) : (
          <p className="whitespace-pre-wrap">{message.text}</p>
        )}
      </div>
    </div>
  );
}

import { useState } from "react";

interface Props {
  onSubmit: (text: string) => void;
  disabled: boolean;
}

export default function ChatInput({ onSubmit, disabled }: Props) {
  const [text, setText] = useState("");

  const handleSubmit = () => {
    const trimmed = text.trim();
    if (!trimmed || disabled) return;
    onSubmit(trimmed);
    setText("");
  };

  return (
    <div className="px-5 py-3 border-t border-[var(--color-border)]">
      <div className="flex gap-2">
        <input
          type="text"
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && handleSubmit()}
          placeholder='试试说"想听点轻松的"...'
          disabled={disabled}
          className="flex-1 bg-[var(--color-bubble-user)] text-[var(--color-text-primary)] text-sm rounded-lg px-4 py-2.5 outline-none border border-[var(--color-border)] focus:border-[var(--color-accent)] transition placeholder:text-[var(--color-text-secondary)]/60 disabled:opacity-50"
        />
        <button
          onClick={handleSubmit}
          disabled={disabled || !text.trim()}
          className="bg-[var(--color-accent)] text-white rounded-lg px-4 py-2.5 text-sm font-medium hover:bg-[var(--color-accent-dim)] transition disabled:opacity-40 disabled:cursor-not-allowed shrink-0"
        >
          发送
        </button>
      </div>
    </div>
  );
}

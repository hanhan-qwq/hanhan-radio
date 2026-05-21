import type { Message } from "../types";
import ChatMessage from "./ChatMessage";

interface Props {
  messages: Message[];
}

export default function ChatArea({ messages }: Props) {
  return (
    <div className="flex-1 overflow-y-auto px-5 py-3 space-y-3 min-h-0">
      {messages.length === 0 ? (
        <div className="flex flex-col items-center justify-center h-full text-center px-4">
          <span className="text-3xl mb-3 opacity-60">🎧</span>
          <p className="text-sm text-[var(--color-text-secondary)]">
            用自然语言告诉 AI 你想听什么
          </p>
          <p className="text-xs text-[var(--color-text-secondary)] mt-1 opacity-60">
            比如"想听点轻松的"、"来首摇滚"、"推荐一首周杰伦的歌"
          </p>
        </div>
      ) : (
        messages.map((msg) => <ChatMessage key={msg.id} message={msg} />)
      )}
    </div>
  );
}

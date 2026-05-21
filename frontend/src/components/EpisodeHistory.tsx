import type { Episode } from "../types";

interface Props {
  episodes: Episode[];
  currentId: string | null;
  onSelect: (episode: Episode) => void;
}

export default function EpisodeHistory({ episodes, currentId, onSelect }: Props) {
  if (episodes.length === 0) {
    return (
      <div className="px-5 py-4 border-b border-[var(--color-border)]">
        <p className="text-xs text-[var(--color-text-secondary)] text-center">
          暂无历史记录
        </p>
      </div>
    );
  }

  return (
    <div className="border-t border-[var(--color-border)]">
      <p className="px-5 py-2 text-xs font-medium text-[var(--color-text-secondary)] uppercase tracking-wide">
        历史记录
      </p>
      <div className="max-h-40 overflow-y-auto">
        {episodes.map((ep) => {
          const seg = ep.segments?.[0];
          const isActive = ep.id === currentId;

          return (
            <button
              key={ep.id}
              onClick={() => onSelect(ep)}
              className={`w-full text-left px-5 py-2.5 flex items-center gap-3 transition hover:bg-white/5 ${
                isActive ? "bg-[var(--color-accent)]/10 border-l-2 border-[var(--color-accent)]" : "border-l-2 border-transparent"
              }`}
            >
              <span className="text-xs shrink-0">
                {ep.status === "done" ? "🎵" : "⏳"}
              </span>
              <div className="min-w-0 flex-1">
                <p className="text-sm text-[var(--color-text-primary)] truncate">
                  {seg ? `${seg.title} - ${seg.artist}` : ep.prompt}
                </p>
                <p className="text-xs text-[var(--color-text-secondary)] truncate">
                  {seg?.segue?.slice(0, 40) || ep.status}
                </p>
              </div>
              <span
                className={`text-xs shrink-0 px-1.5 py-0.5 rounded ${
                  ep.status === "done"
                    ? "bg-green-500/20 text-green-400"
                    : ep.status === "failed"
                      ? "bg-red-500/20 text-red-400"
                      : "bg-blue-500/20 text-blue-400"
                }`}
              >
                {ep.status === "done" ? "完成" : ep.status === "processing" ? "生成中" : ep.status}
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}

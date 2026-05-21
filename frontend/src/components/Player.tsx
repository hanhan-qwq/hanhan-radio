import { useEffect, useRef, useState } from "react";
import type { Episode } from "../types";

interface Props {
  episode: Episode | null;
}

function formatTime(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${s.toString().padStart(2, "0")}`;
}

export default function Player({ episode }: Props) {
  const audioRef = useRef<HTMLAudioElement>(null);
  const [playing, setPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [volume, setVolume] = useState(1);

  useEffect(() => {
    if (episode?.status === "done" && episode.audio_url && audioRef.current) {
      audioRef.current.load();
      audioRef.current.play().catch(() => {});
    }
  }, [episode?.id, episode?.status]);

  useEffect(() => {
    const el = audioRef.current;
    if (!el) return;

    const onTime = () => setCurrentTime(el.currentTime);
    const onDuration = () => setDuration(el.duration);
    const onPlay = () => setPlaying(true);
    const onPause = () => setPlaying(false);
    const onEnded = () => setPlaying(false);

    el.addEventListener("timeupdate", onTime);
    el.addEventListener("loadedmetadata", onDuration);
    el.addEventListener("play", onPlay);
    el.addEventListener("pause", onPause);
    el.addEventListener("ended", onEnded);

    return () => {
      el.removeEventListener("timeupdate", onTime);
      el.removeEventListener("loadedmetadata", onDuration);
      el.removeEventListener("play", onPlay);
      el.removeEventListener("pause", onPause);
      el.removeEventListener("ended", onEnded);
    };
  }, []);

  const togglePlay = () => {
    if (!audioRef.current) return;
    if (playing) {
      audioRef.current.pause();
    } else {
      audioRef.current.play().catch(() => {});
    }
  };

  const seek = (e: React.ChangeEvent<HTMLInputElement>) => {
    const t = parseFloat(e.target.value);
    if (audioRef.current) {
      audioRef.current.currentTime = t;
      setCurrentTime(t);
    }
  };

  const changeVolume = (e: React.ChangeEvent<HTMLInputElement>) => {
    const v = parseFloat(e.target.value);
    setVolume(v);
    if (audioRef.current) audioRef.current.volume = v;
  };

  const segment = episode?.segments?.[0];
  const hasAudio = episode?.status === "done" && episode.audio_url;
  const isLoading = episode && episode.status !== "done" && episode.status !== "failed";

  return (
    <div className="px-5 py-4 border-b border-[var(--color-border)]">
      <audio ref={audioRef} src={episode?.audio_url} preload="auto" />

      {/* Now playing info */}
      <div className="flex items-center gap-4 mb-3">
        <div className="w-12 h-12 rounded-lg bg-[var(--color-accent)]/20 flex items-center justify-center shrink-0">
          {isLoading ? (
            <span className="animate-pulse-soft text-lg">⏳</span>
          ) : hasAudio ? (
            <span className={`text-xl ${playing ? "animate-pulse-soft" : ""}`}>
              🎵
            </span>
          ) : (
            <span className="text-xl opacity-40">🎵</span>
          )}
        </div>

        <div className="min-w-0 flex-1">
          {hasAudio && segment ? (
            <>
              <p className="text-sm font-medium text-[var(--color-text-primary)] truncate">
                {segment.title}
              </p>
              <p className="text-xs text-[var(--color-text-secondary)] truncate">
                {segment.artist}
              </p>
            </>
          ) : isLoading ? (
            <>
              <p className="text-sm font-medium text-[var(--color-text-secondary)]">
                {episode?.status === "pending" ? "正在准备..." : "正在生成..."}
              </p>
              <p className="text-xs text-[var(--color-text-secondary)] truncate">
                AI 正在为您选歌
              </p>
            </>
          ) : (
            <>
              <p className="text-sm text-[var(--color-text-secondary)]">
                暂无播放内容
              </p>
              <p className="text-xs text-[var(--color-text-secondary)]">
                输入心情，开始您的电台之旅
              </p>
            </>
          )}
        </div>

        {hasAudio && (
          <span className="text-xs text-[var(--color-text-secondary)] tabular-nums shrink-0">
            {formatTime(currentTime)} / {formatTime(duration || episode.duration || 0)}
          </span>
        )}
      </div>

      {/* Progress bar */}
      {hasAudio && (
        <div className="mb-3">
          <input
            type="range"
            min={0}
            max={duration || 0}
            value={currentTime}
            onChange={seek}
            className="w-full"
          />
        </div>
      )}

      {/* Controls */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <button
            onClick={togglePlay}
            disabled={!hasAudio}
            className={`w-9 h-9 rounded-full flex items-center justify-center text-sm transition ${
              hasAudio
                ? "bg-[var(--color-accent)] text-white hover:bg-[var(--color-accent-dim)]"
                : "bg-[var(--color-bubble-user)] text-[var(--color-text-secondary)] cursor-not-allowed"
            }`}
          >
            {playing ? "⏸" : "▶"}
          </button>

          {/* Volume */}
          <div className="flex items-center gap-2">
            <span className="text-xs text-[var(--color-text-secondary)]">
              {volume > 0 ? "🔊" : "🔇"}
            </span>
            <input
              type="range"
              min={0}
              max={1}
              step={0.05}
              value={volume}
              onChange={changeVolume}
              className="w-16"
            />
          </div>
        </div>

        {/* Episode status badge */}
        {episode && episode.status !== "done" && (
          <span
            className={`text-xs px-2 py-0.5 rounded-full ${
              episode.status === "failed"
                ? "bg-red-500/20 text-red-400"
                : "bg-blue-500/20 text-blue-400"
            }`}
          >
            {episode.status === "pending"
              ? "等待中"
              : episode.status === "processing"
                ? "生成中"
                : "失败"}
          </span>
        )}

        {hasAudio && episode.duration && (
          <span className="text-xs text-[var(--color-text-secondary)]">
            {Math.floor(episode.duration / 60)}:
            {(episode.duration % 60).toString().padStart(2, "0")}
          </span>
        )}
      </div>
    </div>
  );
}

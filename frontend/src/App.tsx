import { useCallback, useEffect, useState } from "react";
import { createEpisode, listEpisodes } from "./api/client";
import Header from "./components/Header";
import Player from "./components/Player";
import ChatArea from "./components/ChatArea";
import ChatInput from "./components/ChatInput";
import EpisodeHistory from "./components/EpisodeHistory";
import { useEpisodePolling } from "./hooks/useEpisodePolling";
import type { Episode, Message } from "./types";

let msgId = 0;
function nextId() {
  return String(++msgId);
}

export default function App() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [episodes, setEpisodes] = useState<Episode[]>([]);
  const [currentEpisodeId, setCurrentEpisodeId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const { episode: polledEpisode, error: pollError } =
    useEpisodePolling(currentEpisodeId);

  // Update assistant message when polling status changes
  useEffect(() => {
    if (!polledEpisode || !currentEpisodeId) return;

    setMessages((prev) =>
      prev.map((msg) => {
        if (msg.episodeId !== currentEpisodeId || msg.role !== "assistant") return msg;

        if (polledEpisode.status === "done") {
          const segue = polledEpisode.segments?.[0]?.segue || "";
          const title = polledEpisode.segments?.[0]?.title || "";
          const artist = polledEpisode.segments?.[0]?.artist || "";
          return {
            ...msg,
            text: segue || `为您播放 ${title} - ${artist}`,
            episodeStatus: "done",
          };
        }

        return { ...msg, episodeStatus: polledEpisode.status };
      })
    );
  }, [polledEpisode, currentEpisodeId]);

  // Handle polling error
  useEffect(() => {
    if (!pollError || !currentEpisodeId) return;

    setMessages((prev) =>
      prev.map((msg) => {
        if (msg.episodeId !== currentEpisodeId || msg.role !== "assistant") return msg;
        return { ...msg, text: pollError, episodeStatus: "failed" };
      })
    );
    setIsLoading(false);
  }, [pollError, currentEpisodeId]);

  // Refresh episode list when a new episode completes
  useEffect(() => {
    if (polledEpisode?.status === "done") {
      setIsLoading(false);
      listEpisodes().then(setEpisodes).catch(() => {});
    }
  }, [polledEpisode?.status]);

  // Load episode history on mount
  useEffect(() => {
    listEpisodes().then(setEpisodes).catch(() => {});
  }, []);

  const handleSubmit = useCallback(
    async (prompt: string) => {
      setIsLoading(true);

      const userMsg: Message = {
        id: nextId(),
        role: "user",
        text: prompt,
      };

      const assistantMsg: Message = {
        id: nextId(),
        role: "assistant",
        text: "",
        episodeStatus: "pending",
      };

      setMessages((prev) => [...prev, userMsg, assistantMsg]);

      try {
        const ep = await createEpisode(prompt);
        setMessages((prev) =>
          prev.map((msg) => {
            if (msg.id === assistantMsg.id) {
              return { ...msg, episodeId: ep.id };
            }
            return msg;
          })
        );
        setCurrentEpisodeId(ep.id);
      } catch (e) {
        setMessages((prev) =>
          prev.map((msg) => {
            if (msg.id === assistantMsg.id) {
              return {
                ...msg,
                text: e instanceof Error ? e.message : "提交失败",
                episodeStatus: "failed",
              };
            }
            return msg;
          })
        );
        setIsLoading(false);
      }
    },
    []
  );

  const handleHistorySelect = useCallback(
    (ep: Episode) => {
      setCurrentEpisodeId(ep.id);

      // If this episode isn't in messages yet, add it
      setMessages((prev) => {
        const alreadyHas = prev.some((m) => m.episodeId === ep.id && m.role === "user");
        if (alreadyHas) return prev;

        const segue = ep.segments?.[0]?.segue || "";
        const title = ep.segments?.[0]?.title || "";
        const artist = ep.segments?.[0]?.artist || "";

        return [
          ...prev,
          { id: nextId(), role: "user", text: ep.prompt, episodeId: ep.id },
          {
            id: nextId(),
            role: "assistant",
            text: segue || `${title} - ${artist}`,
            episodeId: ep.id,
            episodeStatus: ep.status,
          },
        ];
      });
    },
    []
  );

  const currentEpisode =
    episodes.find((e) => e.id === currentEpisodeId) || polledEpisode || null;

  return (
    <div className="w-full max-w-lg bg-[var(--color-player-bg)] rounded-2xl shadow-2xl overflow-hidden border border-[var(--color-border)] flex flex-col max-h-[90vh]">
      <Header />
      <Player episode={currentEpisode} />
      <ChatArea messages={messages} />
      <ChatInput onSubmit={handleSubmit} disabled={isLoading} />
      <EpisodeHistory
        episodes={episodes}
        currentId={currentEpisodeId}
        onSelect={handleHistorySelect}
      />
    </div>
  );
}

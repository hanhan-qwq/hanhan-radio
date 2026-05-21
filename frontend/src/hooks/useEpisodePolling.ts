import { useEffect, useRef, useState } from "react";
import { getEpisode } from "../api/client";
import type { Episode } from "../types";

interface PollResult {
  episode: Episode | null;
  timedOut: boolean;
  error: string | null;
}

export function useEpisodePolling(episodeId: string | null): PollResult {
  const [episode, setEpisode] = useState<Episode | null>(null);
  const [timedOut, setTimedOut] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const startTime = useRef<number>(0);
  const timerRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined);

  useEffect(() => {
    if (!episodeId) {
      setEpisode(null);
      setTimedOut(false);
      setError(null);
      return;
    }

    startTime.current = Date.now();
    setEpisode(null);
    setTimedOut(false);
    setError(null);

    const poll = async () => {
      try {
        const ep = await getEpisode(episodeId);

        if (ep.status === "failed") {
          setEpisode(ep);
          setError(ep.error || "生成失败");
          clearInterval(timerRef.current);
          return;
        }

        setEpisode(ep);

        if (ep.status === "done") {
          clearInterval(timerRef.current);
          return;
        }

        if (Date.now() - startTime.current > 120_000) {
          setTimedOut(true);
          setError("生成超时，请重试");
          clearInterval(timerRef.current);
        }
      } catch (e) {
        setError(e instanceof Error ? e.message : "未知错误");
        clearInterval(timerRef.current);
      }
    };

    poll();
    timerRef.current = setInterval(poll, 1000);

    return () => clearInterval(timerRef.current);
  }, [episodeId]);

  return { episode, timedOut, error };
}

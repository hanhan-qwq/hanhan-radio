import type { ApiEnvelope, Episode, EpisodeListData } from "../types";

const BASE = "/api/v1";

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, options);
  const json: ApiEnvelope<T> = await res.json();
  if (json.code !== 0) {
    throw new Error(json.message);
  }
  return json.data;
}

export function createEpisode(prompt: string): Promise<Episode> {
  return request<Episode>(`${BASE}/episodes`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ prompt }),
  });
}

export function getEpisode(id: string): Promise<Episode> {
  return request<Episode>(`${BASE}/episodes/${id}`);
}

export function listEpisodes(): Promise<Episode[]> {
  return request<EpisodeListData>(`${BASE}/episodes`).then(
    (data) => data.episodes
  );
}

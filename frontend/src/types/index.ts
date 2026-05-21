export interface SongSegment {
  title: string;
  artist: string;
  segue?: string;
}

export interface Episode {
  id: string;
  prompt: string;
  status: "pending" | "processing" | "done" | "failed";
  created_at: string;
  audio_url?: string;
  duration?: number;
  segments?: SongSegment[];
  message?: string;
  error?: string;
}

export interface EpisodeListData {
  episodes: Episode[];
}

export interface ApiEnvelope<T> {
  code: number;
  message: string;
  data: T;
}

export interface Message {
  id: string;
  role: "user" | "assistant";
  text: string;
  episodeId?: string;
  episodeStatus?: Episode["status"];
}

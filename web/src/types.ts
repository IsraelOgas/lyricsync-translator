export interface TrackInfo {
  artist: string;
  title: string;
  album?: string;
  duration_ms: number;
}

export interface TrackerEvent {
  type: 'track' | 'status' | 'position';
  track?: TrackInfo;
  status?: string;
  position_ms?: number;
  timestamp: number;
}

export interface LyricLineData {
  id: number;
  time_ms: number | null;
  original: string;
  romanized?: string;
  translated?: string;
}

export interface SongInfo {
  id: string;
  hash_key: string;
  artist: string;
  title: string;
  album?: string;
  duration_ms?: number;
  source: string;
}

export interface LyricsLoadingEvent {
  type: 'lyrics_loading';
}

export interface LyricsEvent {
  type: 'lyrics';
  song: SongInfo;
  lines: LyricLineData[];
  not_found?: boolean;
  translating?: boolean;
}

export interface LyricsErrorEvent {
  type: 'lyrics_error';
  error: string;
  retry: boolean;
}

export interface Settings {
  fontSize: number;
  showRomanization: boolean;
}

export const DEFAULT_SETTINGS: Settings = { fontSize: 22, showRomanization: true };

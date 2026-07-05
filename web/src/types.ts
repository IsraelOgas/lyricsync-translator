export interface TrackInfo {
  artist: string;
  title: string;
  album?: string;
  cover_art_url?: string;
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
  offset_ms: number;
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
  fontFamily: 'sans' | 'serif' | 'mono' | 'rounded';
  lineSpacing: number;
  theme: 'dark-purple' | 'dark-blue' | 'warm-amber' | 'minimal-mono';
  translationColor: string;
  romanizationColor: string;
  targetLang: string;
  cinemaMode: boolean;
  textAlignment: 'left' | 'center' | 'right';
}

/** Subset of stored song for list endpoints — no lyric data. */
export interface SongSummary {
  id: string;
  hash_key: string;
  artist: string;
  title: string;
  album?: string;
  created_at: string;
}

export const DEFAULT_SETTINGS: Settings = {
  fontSize: 22,
  showRomanization: true,
  fontFamily: 'sans',
  lineSpacing: 1.8,
  theme: 'dark-purple',
  translationColor: '#55aa55',
  romanizationColor: '#a0a0ee',
  targetLang: 'es',
  cinemaMode: false,
  textAlignment: 'center',
};

// Global declarations for the Wails desktop environment.
declare global {
  interface Window {
    /** Base URL for API calls. Injected by the Go server via template substitution. */
    __API_BASE__: string;
    /** Wails v2 runtime — only available inside a Wails WebView. */
    runtime?: {
      WindowFullscreen: () => void;
      WindowUnfullscreen: () => void;
    };
  }
}

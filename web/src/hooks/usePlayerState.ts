import { useState, useCallback } from 'react';
import { useSSE } from './useSSE';
import { apiUrl } from '../api';
import type { TrackInfo, LyricLineData } from '../types';

export interface UsePlayerStateReturn {
  track: TrackInfo | null;
  status: string;
  positionMs: number;
  lines: LyricLineData[];
  notFound: boolean;
  fetchingLyrics: boolean;
  translating: boolean;
  paused: boolean;
  lyricsError: string | null;
  offsetMs: number;
  handleTogglePlayPause: () => void;
  handleRetryLyrics: () => void;
  handleUpdateOffset: (offsetMs: number) => void;
}

export function usePlayerState(): UsePlayerStateReturn {
  const [track, setTrack] = useState<TrackInfo | null>(null);
  const [status, setStatus] = useState<string>('no_player');
  const [positionMs, setPositionMs] = useState(0);
  const [lines, setLines] = useState<LyricLineData[]>([]);
  const [notFound, setNotFound] = useState(false);
  const [fetchingLyrics, setFetchingLyrics] = useState(false);
  const [translating, setTranslating] = useState(false);
  const [lyricsError, setLyricsError] = useState<string | null>(null);
  const [offsetMs, setOffsetMs] = useState(0);
  const [songHash, setSongHash] = useState<string | null>(null);

  // Derive paused from player status reported via SSE
  const paused = status !== 'playing';

  const handleTogglePlayPause = useCallback(() => {
    fetch(apiUrl('/api/player/toggle'), { method: 'POST' }).catch(() => {});
  }, []);

  const handleRetryLyrics = useCallback(() => {
    setLyricsError(null);
    setFetchingLyrics(true);
    fetch(apiUrl('/api/player/toggle'), { method: 'POST' }).catch(() => {});
  }, []);

  const handleEvent = useCallback((event: any) => {
    switch (event.type) {
      case 'track':
        if (event.track) {
          setTrack(event.track);
          setLines([]);
          setNotFound(false);
          setFetchingLyrics(true);
          setLyricsError(null);
        }
        break;
      case 'status':
        setStatus(event.status || 'unknown');
        break;
      case 'position':
        if (event.position_ms !== undefined) setPositionMs(event.position_ms);
        break;
      case 'lyrics_loading':
        setFetchingLyrics(true);
        break;
      case 'lyrics':
        setFetchingLyrics(false);
        setLyricsError(null);
        setTranslating(!!event.translating);
        if (event.lines) setLines(event.lines);
        if (event.not_found) setNotFound(true);
        if (event.song) {
          setOffsetMs(event.song.offset_ms ?? 0);
          setSongHash(event.song.hash_key ?? null);
        }
        break;
      case 'lyrics_error':
        setFetchingLyrics(false);
        setLyricsError(event.error || 'Failed to load lyrics');
        break;
      case 'translations':
        setTranslating(false);
        // Merge translations into existing lines
        if (event.lines) {
          const tmap = new Map<number, any>(event.lines.map((l: any) => [l.id, l]));
          setLines(prev => prev.map(line => {
            const update = tmap.get(line.id);
            return update ? { ...line, romanized: update.romanized, translated: update.translated } : line;
          }));
        }
        break;
    }
  }, []);

  useSSE(handleEvent);

  const handleUpdateOffset = useCallback((newOffset: number) => {
    setOffsetMs(newOffset);
    if (songHash) {
      fetch(apiUrl(`/api/songs/${encodeURIComponent(songHash)}/offset`), {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ offset_ms: Math.round(newOffset) }),
      }).catch(() => {});
    }
  }, [songHash]);

  return { track, status, positionMs, lines, notFound, fetchingLyrics, translating, paused, lyricsError, offsetMs, handleTogglePlayPause, handleRetryLyrics, handleUpdateOffset };
}

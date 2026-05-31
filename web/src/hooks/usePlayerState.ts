import { useState, useCallback } from 'react';
import { useSSE } from './useSSE';
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
  handleTogglePlayPause: () => void;
}

export function usePlayerState(): UsePlayerStateReturn {
  const [track, setTrack] = useState<TrackInfo | null>(null);
  const [status, setStatus] = useState<string>('no_player');
  const [positionMs, setPositionMs] = useState(0);
  const [lines, setLines] = useState<LyricLineData[]>([]);
  const [notFound, setNotFound] = useState(false);
  const [fetchingLyrics, setFetchingLyrics] = useState(false);
  const [translating, setTranslating] = useState(false);

  // Derive paused from player status reported via SSE
  const paused = status !== 'playing';

  const handleTogglePlayPause = useCallback(() => {
    fetch('/api/player/toggle', { method: 'POST' }).catch(() => {});
  }, []);

  const handleEvent = useCallback((event: any) => {
    switch (event.type) {
      case 'track':
        if (event.track) {
          setTrack(event.track);
          setLines([]);
          setNotFound(false);
          setFetchingLyrics(true);
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
        setTranslating(!!event.translating);
        if (event.lines) setLines(event.lines);
        if (event.not_found) setNotFound(true);
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

  return { track, status, positionMs, lines, notFound, fetchingLyrics, translating, paused, handleTogglePlayPause };
}

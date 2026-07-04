import { useState, useCallback } from 'react';
import { useSSE } from './useSSE';
import { apiUrl } from '../api';
import type { TrackInfo, LyricLineData, PlayerInfo } from '../types';

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
  activePlayer: string;
  players: PlayerInfo[];
  handleTogglePlayPause: () => void;
  handleRetryLyrics: () => void;
  handleUpdateOffset: (offsetMs: number) => void;
  handleSetPlayer: (playerName: string) => void;
  fetchPlayers: () => void;
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
  const [activePlayer, setActivePlayer] = useState<string>('');
  const [players, setPlayers] = useState<PlayerInfo[]>([]);

  // Derive paused from player status reported via SSE
  const paused = status !== 'playing';

  const handleTogglePlayPause = useCallback(() => {
    fetch(apiUrl('/api/player/toggle'), { method: 'POST' }).catch(() => {});
  }, []);

  const handleRetryLyrics = useCallback(() => {
    setLyricsError(null);
    setFetchingLyrics(true);
    fetch(apiUrl('/api/lyrics/retry'), { method: 'POST' }).catch(() => {});
  }, []);

  const handleSetPlayer = useCallback((playerName: string) => {
    fetch(apiUrl('/api/players/active'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ player: playerName }),
    })
      .then(r => r.json())
      .then(data => {
        if (data.active_player !== undefined) {
          setActivePlayer(data.active_player);
        }
      })
      .catch(() => {});
  }, []);

  const fetchPlayers = useCallback(() => {
    fetch(apiUrl('/api/players'))
      .then(r => r.json())
      .then(data => {
        if (data.players) {
          setPlayers(data.players);
        }
      })
      .catch(() => {});
  }, []);

  const handleEvent = useCallback((event: any) => {
    // Track active player from any event that includes it
    if (event.player_name !== undefined) {
      setActivePlayer(event.player_name);
    }

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

  return { track, status, positionMs, lines, notFound, fetchingLyrics, translating, paused, lyricsError, offsetMs, activePlayer, players, handleTogglePlayPause, handleRetryLyrics, handleUpdateOffset, handleSetPlayer, fetchPlayers };
}

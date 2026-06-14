import { useState, useEffect, useCallback } from 'react';
import type { TrackInfo } from '../types';

const STORAGE_KEY = 'lyricsync:bpm';

interface BPMMap {
  [hashKey: string]: number;
}

function loadBPMMap(): BPMMap {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    return stored ? JSON.parse(stored) : {};
  } catch {
    return {};
  }
}

function saveBPMMap(map: BPMMap): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(map));
  } catch { /* ignore quota errors */ }
}

export function useBPM(track: TrackInfo | null, songHash: string | null): [number | null, (bpm: number | null) => void] {
  const [bpm, setBpmState] = useState<number | null>(null);

  // Hydrate from localStorage on mount or track change
  useEffect(() => {
    if (!track || !songHash) {
      setBpmState(null);
      return;
    }
    const map = loadBPMMap();
    if (map[songHash]) {
      setBpmState(map[songHash]);
    }
  }, [track, songHash]);

  const setBpm = useCallback((newBpm: number | null) => {
    setBpmState(newBpm);
    if (!songHash) return;

    const map = loadBPMMap();
    if (newBpm === null) {
      delete map[songHash];
    } else {
      map[songHash] = newBpm;
    }
    saveBPMMap(map);
  }, [songHash]);

  return [bpm, setBpm];
}
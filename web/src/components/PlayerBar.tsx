import React, { useCallback, useState } from 'react';
import { Play, Pause, SkipBack, SkipForward, Shuffle, Repeat, Repeat1, Volume1, Volume2 } from 'lucide-react';
import type { TrackInfo } from '../types';
import styles from './PlayerBar.module.css';

interface Props {
  track: TrackInfo | null;
  status: string;
  positionMs: number;
}

function formatTime(ms: number): string {
  const totalSec = Math.max(0, Math.floor(ms / 1000));
  const min = Math.floor(totalSec / 60);
  const sec = totalSec % 60;
  return `${min}:${sec.toString().padStart(2, '0')}`;
}

/** Fire-and-forget POST helper — never throws. */
function post(url: string, body?: unknown): void {
  fetch(url, {
    method: 'POST',
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  }).catch(() => {});
}

const iconSize = 20;

export const PlayerBar: React.FC<Props> = ({ track, status, positionMs }) => {
  const durationMs = track?.duration_ms ?? 0;
  const hasDuration = durationMs > 0;
  const progress = hasDuration ? Math.min(1, Math.max(0, positionMs / durationMs)) : 0;
  const isPlaying = status === 'playing';
  const [loopState, setLoopState] = useState<string>('none');

  const handleProgressClick = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    if (!hasDuration) return;
    const rect = e.currentTarget.getBoundingClientRect();
    const ratio = Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width));
    post('/api/player/seek', { position_ms: Math.round(ratio * durationMs) });
  }, [hasDuration, durationMs]);

  const handleLoop = useCallback(async () => {
    try {
      const res = await fetch('/api/player/loop', { method: 'POST' });
      const data = await res.json();
      setLoopState((data.loop ?? 'None').toLowerCase());
    } catch { /* ignore */ }
  }, []);

  return (
    <div className={styles.bar}>
      {/* Progress bar */}
      <div className={styles.progressRow}>
        <span className={styles.time}>{formatTime(positionMs)}</span>
        <div
          className={`${styles.progressTrack} ${!hasDuration ? styles.progressDisabled : ''}`}
          onClick={hasDuration ? handleProgressClick : undefined}
          title={hasDuration ? undefined : 'Duration unavailable'}
        >
          <div
            className={styles.progressFill}
            style={{ width: hasDuration ? `${progress * 100}%` : '0%' }}
          />
        </div>
        <span className={styles.time}>{hasDuration ? formatTime(durationMs) : '--:--'}</span>
      </div>

      {/* Controls */}
      <div className={styles.controls}>
        <button className={styles.btn} onClick={() => post('/api/player/shuffle')} title="Shuffle" aria-label="Shuffle">
          <Shuffle size={iconSize} />
        </button>
        <button className={styles.btn} onClick={() => post('/api/player/previous')} title="Previous" aria-label="Previous">
          <SkipBack size={iconSize} />
        </button>
        <button className={styles.btnPlay} onClick={() => post('/api/player/toggle')} title={isPlaying ? 'Pause' : 'Play'} aria-label={isPlaying ? 'Pause' : 'Play'}>
          {isPlaying ? <Pause size={22} /> : <Play size={22} />}
        </button>
        <button className={styles.btn} onClick={() => post('/api/player/next')} title="Next" aria-label="Next">
          <SkipForward size={iconSize} />
        </button>
        <button className={styles.btn} onClick={handleLoop} title="Loop" aria-label="Loop">
          {loopState === 'track' ? <Repeat1 size={iconSize} /> : <Repeat size={iconSize} />}
        </button>
      </div>

      {/* Volume */}
      <div className={styles.volumeRow}>
        <button className={styles.btnVol} onClick={() => post('/api/player/volume', { delta: -0.05 })} title="Volume down" aria-label="Volume down">
          <Volume1 size={16} />
        </button>
        <button className={styles.btnVol} onClick={() => post('/api/player/volume', { delta: 0.05 })} title="Volume up" aria-label="Volume up">
          <Volume2 size={16} />
        </button>
      </div>
    </div>
  );
};

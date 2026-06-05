import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Play, Pause, SkipBack, SkipForward, Shuffle, Repeat, Repeat1, Volume1, Volume2, VolumeX } from 'lucide-react';
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
  const [vol, setVol] = useState(0.5);
  const [muted, setMuted] = useState(false);
  const prevVol = useRef(0.5);
  const volumeFetched = useRef(false);

  // Fetch initial volume from backend
  useEffect(() => {
    if (volumeFetched.current) return;
    volumeFetched.current = true;
    fetch('/api/player/volume')
      .then(r => r.json())
      .then(data => {
        const v = data.volume ?? 0.5;
        setVol(v);
        prevVol.current = v;
      })
      .catch(() => {});
  }, []);

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

  const toggleMute = useCallback(() => {
    if (muted) {
      // Unmute: restore previous volume
      const restore = prevVol.current > 0 ? prevVol.current : 0.5;
      setVol(restore);
      setMuted(false);
      post('/api/player/volume', { absolute: restore });
    } else {
      // Mute
      prevVol.current = vol;
      setMuted(true);
      post('/api/player/volume', { absolute: 0 });
    }
  }, [muted, vol]);

  const handleVolumeChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const v = parseFloat(e.target.value);
    setVol(v);
    if (muted && v > 0) setMuted(false);
    post('/api/player/volume', { absolute: v });
  }, [muted]);

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
        <button
          className={styles.btnVol}
          onClick={toggleMute}
          title={muted ? 'Unmute' : 'Mute'}
          aria-label={muted ? 'Unmute' : 'Mute'}
        >
          {muted || vol === 0 ? <VolumeX size={18} /> : vol < 0.5 ? <Volume1 size={18} /> : <Volume2 size={18} />}
        </button>
        <input
          type="range"
          className={styles.volumeSlider}
          min="0"
          max="1"
          step="0.01"
          value={muted ? 0 : vol}
          onChange={handleVolumeChange}
          aria-label="Volume"
        />
      </div>
    </div>
  );
};

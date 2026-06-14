import React, { useCallback, useState } from 'react';
import type { TrackInfo } from '../types';
import styles from './CinemaProgressBar.module.css';

interface Props {
  track: TrackInfo | null;
  positionMs: number;
}

function formatTime(ms: number): string {
  const totalSec = Math.max(0, Math.floor(ms / 1000));
  const min = Math.floor(totalSec / 60);
  const sec = totalSec % 60;
  return `${min}:${sec.toString().padStart(2, '0')}`;
}

function post(url: string, body?: unknown): void {
  fetch(url, {
    method: 'POST',
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  }).catch(() => {});
}

export const CinemaProgressBar: React.FC<Props> = ({ track, positionMs }) => {
  const durationMs = track?.duration_ms ?? 0;
  const hasDuration = durationMs > 0;
  const progress = hasDuration ? Math.min(1, Math.max(0, positionMs / durationMs)) : 0;
  const [hovering, setHovering] = useState(false);

  const handleClick = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    if (!hasDuration) return;
    const rect = e.currentTarget.getBoundingClientRect();
    const ratio = Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width));
    post('/api/player/seek', { position_ms: Math.round(ratio * durationMs) });
  }, [hasDuration, durationMs]);

  return (
    <div
      className={`${styles.bar} ${hovering ? styles.barExpanded : ''}`}
      onMouseEnter={() => setHovering(true)}
      onMouseLeave={() => setHovering(false)}
    >
      {hovering && (
        <div className={styles.timeLabels}>
          <span className={styles.time}>{formatTime(positionMs)}</span>
          <span className={styles.time}>{hasDuration ? formatTime(durationMs) : '--:--'}</span>
        </div>
      )}
      <div
        className={`${styles.track} ${!hasDuration ? styles.trackDisabled : ''}`}
        onClick={hasDuration ? handleClick : undefined}
      >
        <div
          className={styles.fill}
          style={{ width: hasDuration ? `${progress * 100}%` : '0%' }}
        />
      </div>
    </div>
  );
};

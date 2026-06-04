import React, { useRef, useEffect, useState } from 'react';
import type { LyricLineData } from '../types';
import styles from './LyricsViewer.module.css';

interface Props {
  lines: LyricLineData[];
  positionMs: number;
  offsetMs: number;
  paused: boolean;
  notFound?: boolean;
  fetchingLyrics?: boolean;
  translating?: boolean;
  lyricsError?: string | null;
  onRetry?: () => void;
  onUpdateOffset?: (offsetMs: number) => void;
  showRomanization?: boolean;
}

const LAST_LINE_DURATION_MS = 3500;

export const LyricsViewer: React.FC<Props> = ({ lines, positionMs, offsetMs, paused, notFound, fetchingLyrics, translating, lyricsError, onRetry, onUpdateOffset, showRomanization = true }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [activeIdx, setActiveIdx] = useState(-1);
  const [lineProgress, setLineProgress] = useState(0);

  // Apply offset to effective position
  const effectiveMs = positionMs + offsetMs;

  useEffect(() => {
    if (lines.length === 0 || paused) return;

    const syncedLines = lines
      .map((l, i) => ({ ...l, origIdx: i }))
      .filter((l) => l.time_ms !== null)
      .sort((a, b) => (a.time_ms ?? 0) - (b.time_ms ?? 0));

    if (syncedLines.length === 0) return;

    let lo = 0;
    let hi = syncedLines.length - 1;
    let best = 0;
    while (lo <= hi) {
      const mid = Math.floor((lo + hi) / 2);
      const t = syncedLines[mid].time_ms ?? 0;
      if (t <= effectiveMs) {
        best = mid;
        lo = mid + 1;
      } else {
        hi = mid - 1;
      }
    }

    const activeLine = syncedLines[best];
    setActiveIdx(activeLine.origIdx);

    const startMs = activeLine.time_ms ?? 0;
    const nextLine = best + 1 < syncedLines.length ? syncedLines[best + 1] : null;
    const endMs = nextLine?.time_ms ?? startMs + LAST_LINE_DURATION_MS;
    const duration = endMs - startMs;

    if (duration > 0) {
      setLineProgress(Math.min(1, Math.max(0, (effectiveMs - startMs) / duration)));
    } else {
      setLineProgress(0);
    }
  }, [effectiveMs, lines, paused]);

  useEffect(() => {
    if (paused) return;
    const container = containerRef.current;
    if (!container) return;
    const activeEl = container.querySelector('[data-active="true"]');
    if (activeEl) {
      activeEl.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  }, [activeIdx, paused]);

  if (lines.length === 0) {
    if (lyricsError && !fetchingLyrics) {
      return (
        <div className={styles.empty}>
          <p className={styles.emptyText}>{lyricsError}</p>
          <button className={styles.retryBtn} onClick={onRetry}>Retry</button>
        </div>
      );
    }
    if (fetchingLyrics) {
      return (
        <div className={styles.empty}>
          <div className={styles.spinner} />
          <p className={styles.emptyText}>Loading lyrics...</p>
        </div>
      );
    }
    return (
      <div className={styles.empty}>
        {notFound ? (
          <>
            <p className={styles.emptyText}>Lyrics not found</p>
            <p className={styles.emptySub}>Try another song</p>
          </>
        ) : (
          <>
            <p className={styles.emptyText}>No lyrics loaded</p>
            <p className={styles.emptySub}>Start playing a song</p>
          </>
        )}
      </div>
    );
  }

  return (
    <div ref={containerRef} className={`${styles.container} ${paused ? styles.containerPaused : ''}`}>
      {paused && (
        <div className={styles.pauseBanner}>⏸ PAUSED</div>
      )}

      {/* Offset slider */}
      <div className={styles.offsetBar}>
        <span className={styles.offsetLabel}>Sync</span>
        <input
          type="range"
          className={styles.offsetSlider}
          min={-5000}
          max={5000}
          step={100}
          value={offsetMs}
          onChange={e => onUpdateOffset?.(Number(e.target.value))}
        />
        <span className={styles.offsetValue}>{offsetMs > 0 ? '+' : ''}{(offsetMs / 1000).toFixed(1)}s</span>
      </div>

      {lines.map((line, idx) => {
        const isActive = idx === activeIdx && !paused;
        const pendingTranslation = translating && !line.romanized && !line.translated;
        return (
          <div
            key={line.id || idx}
            data-active={isActive}
            className={`${styles.line} ${isActive ? styles.lineActive : styles.lineInactive}`}
          >
            <p
              className={`${styles.original} ${isActive ? styles.karaoke : ''}`}
              style={isActive ? { '--karaoke-progress': lineProgress } as React.CSSProperties : undefined}
            >
              {line.original}
            </p>
            {showRomanization !== false && line.romanized && (
              <p className={styles.romanized}>{line.romanized}</p>
            )}
            {line.translated && (
              <p className={styles.translated}>{line.translated}</p>
            )}
            {pendingTranslation && (
              <p className={styles.translatingHint}>translating...</p>
            )}
          </div>
        );
      })}
    </div>
  );
};

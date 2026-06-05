import React, { useRef, useEffect, useState } from 'react';
import { Pause, RotateCcw } from 'lucide-react';
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
  showRomanization?: boolean;
}


export const LyricsViewer: React.FC<Props> = ({ lines, positionMs, offsetMs, paused, notFound, fetchingLyrics, translating, lyricsError, onRetry, showRomanization = true }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [activeIdx, setActiveIdx] = useState(-1);

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

    setActiveIdx(syncedLines[best].origIdx);
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
          <button className={styles.retryBtn} onClick={onRetry}><RotateCcw size={14} /> Retry</button>
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

  const handleLineClick = (timeMs: number) => {
    fetch('/api/player/seek', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ position_ms: timeMs }),
    }).catch(() => {}); // silent — playerctl errors are non-critical
  };

  return (
    <div ref={containerRef} className={`${styles.container} ${paused ? styles.containerPaused : ''}`}>
      {paused && (
        <div className={styles.pauseBanner}><Pause size={14} /> PAUSED</div>
      )}

      {lines.map((line, idx) => {
        const isActive = idx === activeIdx && !paused;
        const isClickable = line.time_ms != null;
        const isInstrumental = !line.original.trim();
        const pendingTranslation = !isInstrumental && translating && !line.romanized && !line.translated;
        return (
          <div
            key={line.id || idx}
            data-active={isActive}
            className={`${styles.line} ${isActive ? styles.lineActive : styles.lineInactive} ${isClickable ? styles.clickable : ''}`}
            onClick={isClickable ? () => handleLineClick(line.time_ms!) : undefined}
            title={isClickable ? (isInstrumental ? 'Instrumental' : 'Click to jump to this verse') : undefined}
          >
            {isInstrumental ? (
              <p className={styles.instrumental}>— ♪ —</p>
            ) : (
              <p className={styles.original}>
                {line.original}
              </p>
            )}
            {!isInstrumental && showRomanization !== false && line.romanized && (
              <p className={styles.romanized}>{line.romanized}</p>
            )}
            {!isInstrumental && line.translated && (
              <p className={styles.translated}>{line.translated}</p>
            )}
            {pendingTranslation && (
              <div className={styles.translatingShimmer} />
            )}
          </div>
        );
      })}
    </div>
  );
};

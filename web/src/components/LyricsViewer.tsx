import React, { useRef, useEffect, useState, useCallback } from 'react';
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
  /** When true, suppresses position sync and click-to-seek for static display. */
  staticMode?: boolean;
}


export const LyricsViewer: React.FC<Props> = ({ lines, positionMs, offsetMs, paused, notFound, fetchingLyrics, translating, lyricsError, onRetry, showRomanization = true, staticMode = false }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const activeLineRef = useRef<HTMLDivElement>(null);
  const [activeIdx, setActiveIdx] = useState(-1);
  const [toastVisible, setToastVisible] = useState(false);
  const toastTimer = useRef<number>(0);
  const prevActiveIdx = useRef(-1);

  // Show toast when error arrives, auto-dismiss after 6s.
  useEffect(() => {
    if (lyricsError) {
      setToastVisible(true);
      clearTimeout(toastTimer.current);
      toastTimer.current = setTimeout(() => setToastVisible(false), 6000);
    } else {
      setToastVisible(false);
    }
    return () => clearTimeout(toastTimer.current);
  }, [lyricsError]);

  // Apply offset to effective position
  const effectiveMs = positionMs + offsetMs;

  // Synced lines cache — only recomputes when lines change
  const syncedLinesRef = useRef<{ idx: number; timeMs: number }[]>([]);

  useEffect(() => {
    syncedLinesRef.current = lines
      .map((l, i) => ({ idx: i, timeMs: l.time_ms ?? 0 }))
      .filter(l => l.timeMs > 0)
      .sort((a, b) => a.timeMs - b.timeMs);
  }, [lines]);

  // Update active line + karaoke progress WITHOUT triggering React re-render
  useEffect(() => {
    if (staticMode || syncedLinesRef.current.length === 0 || paused) return;

    const synced = syncedLinesRef.current;
    let lo = 0;
    let hi = synced.length - 1;
    let best = 0;
    while (lo <= hi) {
      const mid = Math.floor((lo + hi) / 2);
      if (synced[mid].timeMs <= effectiveMs) {
        best = mid;
        lo = mid + 1;
      } else {
        hi = mid - 1;
      }
    }

    const newActiveIdx = synced[best].idx;

    // Only call setState if the active line actually changed
    if (newActiveIdx !== prevActiveIdx.current) {
      prevActiveIdx.current = newActiveIdx;
      setActiveIdx(newActiveIdx);
    }

    // Update karaoke progress directly on DOM (bypass React render)
    if (activeLineRef.current) {
      const lineStart = synced[best].timeMs;
      const nextLine = synced.find((_l, i) => i > best);
      const lineEnd = nextLine?.timeMs ?? (lineStart + 4000);
      const duration = Math.max(lineEnd - lineStart, 1);
      const progress = Math.min(1, Math.max(0, (effectiveMs - lineStart) / duration));
      activeLineRef.current.style.setProperty('--karaoke-progress', String(progress));
    }
  }, [effectiveMs, paused, staticMode]);

  // Scroll to active line
  useEffect(() => {
    if (staticMode || paused) return;
    const container = containerRef.current;
    if (!container) return;
    const activeEl = container.querySelector('[data-active="true"]');
    if (activeEl) {
      activeEl.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  }, [activeIdx, paused]);

  // Assign ref callback to the active line element
  const lineRefCallback = useCallback((node: HTMLDivElement | null) => {
    activeLineRef.current = node;
  }, []);

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
    if (staticMode) return;
    fetch('/api/player/seek', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ position_ms: timeMs }),
    }).catch(() => {}); // silent — playerctl errors are non-critical
  };

  const hasRomanization = lines.some(l => l.romanized);

  return (
    <div ref={containerRef} className={`${styles.container} ${paused ? styles.containerPaused : ''}`} data-has-romanization={hasRomanization ? 'true' : 'false'}>
      {paused && (
        <div className={styles.pauseBanner}><Pause size={14} /> PAUSED</div>
      )}

      {toastVisible && lyricsError && (
        <div className={styles.toast}>
          <span className={styles.toastText}>{lyricsError}</span>
          <button className={styles.toastBtn} onClick={onRetry}>Retry</button>
        </div>
      )}

      {lines.map((line, idx) => {
        const isActive = idx === activeIdx && !paused;
        const isClickable = line.time_ms != null;
        const isInstrumental = !line.original.trim();
        const pendingTranslation = !isInstrumental && translating && !line.romanized && !line.translated;
        return (
          <div
            key={line.id || idx}
            ref={isActive ? lineRefCallback : undefined}
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

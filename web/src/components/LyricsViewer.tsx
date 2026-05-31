import React, { useRef, useEffect, useState } from 'react';
import type { LyricLineData } from '../types';
import styles from './LyricsViewer.module.css';

interface Props {
  lines: LyricLineData[];
  positionMs: number;
  paused: boolean;
  notFound?: boolean;
  fetchingLyrics?: boolean;
  translating?: boolean;
}

export const LyricsViewer: React.FC<Props> = ({ lines, positionMs, paused, notFound, fetchingLyrics, translating }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [activeIdx, setActiveIdx] = useState(-1);

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
      if (t <= positionMs) {
        best = mid;
        lo = mid + 1;
      } else {
        hi = mid - 1;
      }
    }

    setActiveIdx(syncedLines[best].origIdx);
  }, [positionMs, lines, paused]);

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
    if (fetchingLyrics) {
      return (
        <div className={styles.empty}>
          <div className={styles.skeletonLine} />
          <div className={styles.skeletonLine} />
          <div className={styles.skeletonLine} />
          <div className={styles.skeletonLine} />
          <div className={styles.skeletonLine} />
          <div className={styles.skeletonLine} />
          <div className={styles.skeletonLine} />
          <div className={styles.skeletonLine} />
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
      {lines.map((line, idx) => {
        const isActive = idx === activeIdx && !paused;
        const pendingTranslation = translating && !line.romanized && !line.translated;
        return (
          <div
            key={line.id || idx}
            data-active={isActive}
            className={`${styles.line} ${isActive ? styles.lineActive : styles.lineInactive}`}
          >
            <p className={styles.original}>{line.original}</p>
            {line.romanized && (
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

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
  karaokeMode?: boolean;
  /** When true, suppresses position sync and click-to-seek for static display. */
  staticMode?: boolean;
}

/** Per-word state for karaoke rendering. */
interface WordState {
  text: string;
  progress: number; // 0..1 for current word, -1 pending, 1 done
}

export const LyricsViewer: React.FC<Props> = ({ lines, positionMs, offsetMs, paused, notFound, fetchingLyrics, translating, lyricsError, onRetry, showRomanization = true, karaokeMode = true, staticMode = false }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [activeIdx, setActiveIdx] = useState(-1);
  const [toastVisible, setToastVisible] = useState(false);
  const toastTimer = useRef<number>(0);
  // Word-level karaoke states for the active line.
  const [wordStates, setWordStates] = useState<WordState[]>([]);
  // Legacy line-level progress fallback.
  const [karaokeProgress, setKaraokeProgress] = useState(0);

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

  useEffect(() => {
    if (staticMode || lines.length === 0 || paused) return;

    const syncedLines = lines
      .map((l, i) => ({ ...l, origIdx: i }))
      .filter((l) => l.time_ms != null)
      .sort((a, b) => (a.time_ms ?? 0) - (b.time_ms ?? 0));

    if (syncedLines.length === 0) return;

    // If every timestamp is 0, the lyrics are unsynced (plain text).
    if (syncedLines.every((l) => l.time_ms === 0)) return;

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

    if (karaokeMode) {
      const activeLine = syncedLines[best];

      // Word-level karaoke: calculate progress for each word.
      if (activeLine.words && activeLine.words.length > 0) {
        const states: WordState[] = activeLine.words.map((w) => {
          const duration = Math.max(w.end_ms - w.start_ms, 1);
          const raw = (effectiveMs - w.start_ms) / duration;
          if (raw <= 0) return { text: w.text, progress: -1 };   // pending
          if (raw >= 1) return { text: w.text, progress: 1 };     // done
          return { text: w.text, progress: Math.min(1, Math.max(0, raw)) }; // active
        });
        setWordStates(states);
        setKaraokeProgress(0); // not used in word mode
      } else {
        // Line-level fallback.
        const nextLine = syncedLines.find((l, i) => i > best && l.time_ms != null);
        const lineStart = activeLine.time_ms ?? 0;
        const lineEnd = nextLine?.time_ms ?? (lineStart + 4000);
        const duration = Math.max(lineEnd - lineStart, 1);
        const progress = Math.min(1, Math.max(0, (effectiveMs - lineStart) / duration));
        setKaraokeProgress(progress);
        setWordStates([]);
      }
    }
  }, [effectiveMs, lines, paused, karaokeMode]);

  useEffect(() => {
    if (staticMode || paused) return;
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
    if (staticMode) return;
    fetch('/api/player/seek', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ position_ms: timeMs }),
    }).catch(() => {}); // silent — playerctl errors are non-critical
  };

  const hasRomanization = lines.some(l => l.romanized);
  const hasWordKaraoke = wordStates.length > 0;

  /** Render original text: word spans if word-level data exists, plain text otherwise. */
  const renderOriginal = (line: LyricLineData, isActive: boolean) => {
    // Word-level karaoke on active line.
    if (isActive && hasWordKaraoke) {
      return (
        <p className={styles.original}>
          {wordStates.map((ws, wi) => (
            <span
              key={wi}
              className={
                ws.progress >= 1 ? styles.wordDone :
                ws.progress < 0 ? styles.wordPending :
                styles.wordActive
              }
              style={ws.progress > 0 && ws.progress < 1 ? { '--word-progress': ws.progress } as React.CSSProperties : undefined}
            >
              {ws.text}
            </span>
          ))}
        </p>
      );
    }

    // Word-level on inactive lines: render spans but no progress classes.
    if (line.words && line.words.length > 0) {
      return (
        <p className={styles.original}>
          {line.words.map((w, wi) => (
            <span key={wi}>{w.text}</span>
          ))}
        </p>
      );
    }

    // Plain text fallback.
    return <p className={styles.original}>{line.original}</p>;
  };

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
            data-active={isActive}
            className={`${styles.line} ${isActive ? styles.lineActive : styles.lineInactive} ${isClickable ? styles.clickable : ''}`}
            style={isActive && karaokeMode && !hasWordKaraoke ? { '--karaoke-progress': karaokeProgress } as React.CSSProperties : undefined}
            onClick={isClickable ? () => handleLineClick(line.time_ms!) : undefined}
            title={isClickable ? (isInstrumental ? 'Instrumental' : 'Click to jump to this verse') : undefined}
          >
            {isInstrumental ? (
              <p className={styles.instrumental}>— ♪ —</p>
            ) : (
              renderOriginal(line, isActive)
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

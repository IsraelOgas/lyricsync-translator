import React, { useEffect, useRef, useState } from 'react';
import styles from './BeatVisualizer.module.css';

interface Props {
  bpm: number | null;
  positionMs: number;
  isPlaying: boolean;
}

export const BeatVisualizer: React.FC<Props> = ({ bpm, positionMs, isPlaying }) => {
  const [beatPhase, setBeatPhase] = useState(0);
  const lastBeatRef = useRef<number | null>(null);
  const rafRef = useRef<number | undefined>(undefined);

  useEffect(() => {
    if (!bpm || !isPlaying) {
      setBeatPhase(0);
      lastBeatRef.current = null;
      return;
    }

    const beatIntervalMs = 60000 / bpm;

    const tick = () => {
      const pos = positionMs;
      const currentBeat = Math.floor(pos / beatIntervalMs);

      if (lastBeatRef.current !== null && currentBeat !== lastBeatRef.current) {
        setBeatPhase(1);
        setTimeout(() => setBeatPhase(0), 80);
      }
      lastBeatRef.current = currentBeat;

      rafRef.current = requestAnimationFrame(tick);
    };

    rafRef.current = requestAnimationFrame(tick);
    return () => {
      if (rafRef.current) cancelAnimationFrame(rafRef.current);
    };
  }, [bpm, positionMs, isPlaying]);

  if (!bpm) return null;

  return (
    <div className={styles.container} aria-hidden="true">
      <div
        className={`${styles.beat} ${beatPhase === 1 ? styles.active : ''}`}
        style={{
          '--beat-color': 'var(--color-primary)',
        } as React.CSSProperties}
      />
      <span className={styles.bpmLabel}>{Math.round(bpm)} BPM</span>
    </div>
  );
};
import React, { useEffect, useRef } from 'react';
import styles from './AdvancedVisualizer.module.css';
import type { BeatData } from '../hooks/usePlayerState';

type VisMode = 'bars' | 'wave' | 'circular' | 'pulse';

interface Props {
  bpm: number | null;
  positionMs: number;
  isPlaying: boolean;
  mode: VisMode;
  beat: BeatData;
}

const BAR_COUNT = 32;
const ENERGY_HISTORY_LEN = 30;
const BEAT_THRESHOLD = 1.3;
const MIN_BEAT_GAP_MS = 200;

export const AdvancedVisualizer: React.FC<Props> = ({ bpm, positionMs, isPlaying, mode, beat }) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const rafRef = useRef<number | undefined>(undefined);
  const energyHistoryRef = useRef<number[]>([]);
  const beatPhaseRef = useRef(0);
  const lastBeatMsRef = useRef(0);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const resize = () => {
      const rect = canvas.getBoundingClientRect();
      canvas.width = rect.width * window.devicePixelRatio;
      canvas.height = rect.height * window.devicePixelRatio;
      ctx.scale(window.devicePixelRatio, window.devicePixelRatio);
    };
    resize();
    window.addEventListener('resize', resize);

    const history = energyHistoryRef.current;

    const draw = () => {
      const w = canvas.getBoundingClientRect().width;
      const h = canvas.getBoundingClientRect().height;
      ctx.clearRect(0, 0, w, h);

      if (beat.energy === 0) {
        drawIdle(ctx, w, h);
        rafRef.current = requestAnimationFrame(draw);
        return;
      }

      // Track energy for beat detection and visualization
      history.push(beat.energy);
      if (history.length > ENERGY_HISTORY_LEN) history.shift();

      // Frontend beat detection: energy peak above local average
      const avg = history.reduce((a, b) => a + b, 0) / history.length;
      const now = Date.now();
      const timeSinceLastBeat = now - lastBeatMsRef.current;
      const isBeat = beat.energy > avg * BEAT_THRESHOLD
        && beat.energy > 0.02
        && timeSinceLastBeat > MIN_BEAT_GAP_MS;

      if (isBeat) {
        lastBeatMsRef.current = now;
      }

      // Update beat phase (0.0 to 1.0 between beats)
      if (bpm && bpm > 0) {
        const beatIntervalMs = 60000 / bpm;
        const timeSinceBeat = now - lastBeatMsRef.current;
        beatPhaseRef.current = Math.min(1.0, timeSinceBeat / beatIntervalMs);
      } else {
        // No BPM: use energy directly as phase driver
        beatPhaseRef.current = isBeat ? 0 : Math.min(1.0, beatPhaseRef.current + 0.08);
      }

      const phase = beatPhaseRef.current;
      const energy = beat.energy;

      switch (mode) {
        case 'bars':
          drawBars(ctx, w, h, phase, energy, isBeat);
          break;
        case 'wave':
          drawWave(ctx, w, h, phase, energy, positionMs);
          break;
        case 'circular':
          drawCircular(ctx, w, h, phase, energy, isBeat);
          break;
        case 'pulse':
          drawPulse(ctx, w, h, phase, energy, isBeat);
          break;
      }

      rafRef.current = requestAnimationFrame(draw);
    };

    rafRef.current = requestAnimationFrame(draw);
    return () => {
      window.removeEventListener('resize', resize);
      if (rafRef.current) cancelAnimationFrame(rafRef.current);
    };
  }, [bpm, positionMs, isPlaying, mode, beat]);

  return (
    <div className={styles.container}>
      <canvas ref={canvasRef} className={styles.canvas} />
    </div>
  );
};

function drawIdle(ctx: CanvasRenderingContext2D, w: number, h: number) {
  ctx.strokeStyle = 'rgba(136, 136, 204, 0.15)';
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.moveTo(0, h / 2);
  ctx.lineTo(w, h / 2);
  ctx.stroke();
}

function drawBars(ctx: CanvasRenderingContext2D, w: number, h: number, phase: number, energy: number, isNewBeat: boolean) {
  const barWidth = w / BAR_COUNT;
  const gap = 2;

  for (let i = 0; i < BAR_COUNT; i++) {
    const freqFactor = Math.sin((i / BAR_COUNT) * Math.PI);
    const baseHeight = energy * freqFactor;

    let heightPct: number;
    if (isNewBeat) {
      heightPct = 0.3 + baseHeight * 0.7;
    } else {
      const decay = Math.pow(1 - phase, 2);
      heightPct = Math.max(0.05, (0.3 + baseHeight * 0.7) * decay);
    }

    const barH = heightPct * h * 0.85;
    const x = i * barWidth + gap / 2;
    const y = h - barH;

    const brightness = 0.4 + energy * 0.6;
    ctx.fillStyle = `rgba(136, 136, 204, ${brightness})`;
    ctx.beginPath();
    ctx.roundRect(x, y, barWidth - gap, barH, 2);
    ctx.fill();

    if (energy > 0.3) {
      ctx.shadowColor = `rgba(136, 136, 204, ${energy * 0.5})`;
      ctx.shadowBlur = 8;
      ctx.fill();
      ctx.shadowBlur = 0;
    }
  }
}

function drawWave(ctx: CanvasRenderingContext2D, w: number, h: number, phase: number, energy: number, posMs: number) {
  const amplitude = h * 0.35 * (0.3 + energy * 0.7);
  const timeShift = posMs * 0.002;

  ctx.beginPath();

  for (let x = 0; x <= w; x += 2) {
    const freq1 = 0.02 + energy * 0.01;
    const freq2 = 0.045;
    const y1 = Math.sin(x * freq1 + timeShift) * amplitude;
    const y2 = Math.sin(x * freq2 + timeShift * 1.3 + phase * Math.PI * 2) * amplitude * 0.3 * energy;
    const y = h / 2 + y1 + y2;
    if (x === 0) ctx.moveTo(x, y);
    else ctx.lineTo(x, y);
  }

  const gradient = ctx.createLinearGradient(0, 0, w, 0);
  gradient.addColorStop(0, 'rgba(136, 136, 204, 0.6)');
  gradient.addColorStop(0.5, `rgba(136, 136, 204, ${0.5 + energy * 0.5})`);
  gradient.addColorStop(1, 'rgba(136, 136, 204, 0.6)');
  ctx.strokeStyle = gradient;
  ctx.lineWidth = 2;
  ctx.stroke();

  if (energy > 0.2) {
    ctx.shadowColor = `rgba(136, 136, 204, ${energy * 0.6})`;
    ctx.shadowBlur = 10;
    ctx.stroke();
    ctx.shadowBlur = 0;
  }

  ctx.lineTo(w, h);
  ctx.lineTo(0, h);
  ctx.closePath();
  ctx.fillStyle = `rgba(136, 136, 204, ${0.02 + energy * 0.06})`;
  ctx.fill();
}

function drawCircular(ctx: CanvasRenderingContext2D, w: number, h: number, phase: number, energy: number, isNewBeat: boolean) {
  const cx = w / 2;
  const cy = h / 2;
  const baseR = Math.min(w, h) * 0.28;
  const segments = 48;

  for (let i = 0; i < segments; i++) {
    const angle = (i / segments) * Math.PI * 2 - Math.PI / 2;

    const segmentPhase = (phase + i * 0.02) % 1;
    const intensity = energy * (0.5 + Math.sin(segmentPhase * Math.PI * 2) * 0.5);

    const burst = isNewBeat ? 0.3 : 0;
    const len = baseR * (0.3 + intensity * 0.7 + burst);

    const x1 = cx + Math.cos(angle) * baseR;
    const y1 = cy + Math.sin(angle) * baseR;
    const x2 = cx + Math.cos(angle) * (baseR + len);
    const y2 = cy + Math.sin(angle) * (baseR + len);

    ctx.beginPath();
    ctx.moveTo(x1, y1);
    ctx.lineTo(x2, y2);
    ctx.strokeStyle = `rgba(136, 136, 204, ${0.3 + intensity * 0.7})`;
    ctx.lineWidth = 2;
    ctx.stroke();
  }

  const pulseR = baseR * (0.8 + energy * 0.4);
  ctx.beginPath();
  ctx.arc(cx, cy, pulseR, 0, Math.PI * 2);
  ctx.strokeStyle = `rgba(136, 136, 204, ${0.2 + energy * 0.3})`;
  ctx.lineWidth = 1.5;
  ctx.stroke();
}

function drawPulse(ctx: CanvasRenderingContext2D, w: number, h: number, phase: number, energy: number, isNewBeat: boolean) {
  const cx = w / 2;
  const cy = h / 2;
  const maxR = Math.min(w, h) * 0.45;

  const rings = 4;
  for (let i = 0; i < rings; i++) {
    const ringPhase = (phase + i * 0.15) % 1;
    const r = maxR * (0.2 + ringPhase * 0.8) * (0.6 + energy * 0.6);
    const alpha = Math.max(0, 0.5 * (1 - ringPhase) * energy);

    ctx.beginPath();
    ctx.arc(cx, cy, r, 0, Math.PI * 2);
    ctx.strokeStyle = `rgba(136, 136, 204, ${alpha})`;
    ctx.lineWidth = 2;
    ctx.stroke();
  }

  const coreR = maxR * 0.12 * (0.5 + energy * 0.8);
  const gradient = ctx.createRadialGradient(cx, cy, 0, cx, cy, coreR);
  const coreAlpha = isNewBeat ? 0.9 : 0.3 + energy * 0.4;
  gradient.addColorStop(0, `rgba(136, 136, 204, ${coreAlpha})`);
  gradient.addColorStop(1, 'rgba(136, 136, 204, 0)');
  ctx.fillStyle = gradient;
  ctx.beginPath();
  ctx.arc(cx, cy, coreR, 0, Math.PI * 2);
  ctx.fill();
}

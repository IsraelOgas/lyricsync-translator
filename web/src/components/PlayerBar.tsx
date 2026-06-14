import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Play, Pause, SkipBack, SkipForward, Shuffle, Repeat, Repeat1, Volume1, Volume2, VolumeX, Settings, HelpCircle, Music, BarChart3, Activity, Circle, Disc } from 'lucide-react';
import type { TrackInfo } from '../types';
import type { BeatData } from '../hooks/usePlayerState';
import { BeatVisualizer } from './BeatVisualizer';
import { AdvancedVisualizer } from './AdvancedVisualizer';
import { useBPM } from '../hooks/useBPM';
import styles from './PlayerBar.module.css';

type VisMode = 'bars' | 'wave' | 'circular' | 'pulse';
const VIS_MODES: { key: VisMode; icon: React.ReactNode; label: string }[] = [
  { key: 'bars', icon: <BarChart3 size={14} />, label: 'Bars' },
  { key: 'wave', icon: <Activity size={14} />, label: 'Wave' },
  { key: 'circular', icon: <Disc size={14} />, label: 'Circular' },
  { key: 'pulse', icon: <Circle size={14} />, label: 'Pulse' },
];

interface Props {
  track: TrackInfo | null;
  status: string;
  positionMs: number;
  songHash: string | null;
  beat: BeatData;
  onOpenSettings: () => void;
  onOpenHelp: () => void;
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

export const PlayerBar: React.FC<Props> = ({ track, status, positionMs, songHash, beat, onOpenSettings, onOpenHelp }) => {
  const durationMs = track?.duration_ms ?? 0;
  const hasDuration = durationMs > 0;
  const progress = hasDuration ? Math.min(1, Math.max(0, positionMs / durationMs)) : 0;
  const isPlaying = status === 'playing';
  const [loopState, setLoopState] = useState<string>('none');
  const [shuffleOn, setShuffleOn] = useState(false);
  const [vol, setVol] = useState(0.5);
  const [muted, setMuted] = useState(false);
  // Use backend-detected BPM if available, otherwise allow manual override
  const detectedBpm = beat.bpm;
  const [manualBpm, setManualBpm] = useBPM(track, songHash);
  const bpm = detectedBpm || manualBpm;
  const setBpm = setManualBpm;
  const [bpmInput, setBpmInput] = useState('');
  const [visMode, setVisMode] = useState<VisMode>(() => {
    try { return (localStorage.getItem('lyricsync:visMode') as VisMode) || 'bars'; } catch { return 'bars'; }
  });
  const prevVol = useRef(0.5);
  const stateFetched = useRef(false);

  // Fetch initial player state from backend
  useEffect(() => {
    if (stateFetched.current) return;
    stateFetched.current = true;

    fetch('/api/player/volume')
      .then(r => r.json())
      .then(data => {
        const v = data.volume ?? 0.5;
        setVol(v);
        prevVol.current = v;
      })
      .catch(() => {});

    fetch('/api/player/shuffle')
      .then(r => r.json())
      .then(data => setShuffleOn(!!data.shuffle))
      .catch(() => {});

    fetch('/api/player/loop')
      .then(r => r.json())
      .then(data => setLoopState((data.loop ?? 'None').toLowerCase()))
      .catch(() => {});
  }, []);

  const handleProgressClick = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    if (!hasDuration) return;
    const rect = e.currentTarget.getBoundingClientRect();
    const ratio = Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width));
    post('/api/player/seek', { position_ms: Math.round(ratio * durationMs) });
  }, [hasDuration, durationMs]);

  const handleShuffle = useCallback(async () => {
    try {
      await fetch('/api/player/shuffle', { method: 'POST' });
      // Read back the new state to stay in sync
      const res = await fetch('/api/player/shuffle');
      const data = await res.json();
      setShuffleOn(!!data.shuffle);
    } catch { /* ignore */ }
  }, []);

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

  const cycleVisMode = useCallback(() => {
    setVisMode(prev => {
      const idx = VIS_MODES.findIndex(m => m.key === prev);
      const next = VIS_MODES[(idx + 1) % VIS_MODES.length].key;
      try { localStorage.setItem('lyricsync:visMode', next); } catch {}
      return next;
    });
  }, []);

  const tapTimesRef = useRef<number[]>([]);
  const handleTapTempo = useCallback(() => {
    const now = performance.now();
    const taps = tapTimesRef.current;
    taps.push(now);

    // Keep only last 8 taps
    while (taps.length > 8) taps.shift();

    if (taps.length < 2) return;

    // If more than 3 seconds since last tap, reset
    if (now - taps[taps.length - 2] > 3000) {
      taps.length = 1;
      return;
    }

    // Calculate average interval
    let total = 0;
    for (let i = 1; i < taps.length; i++) {
      total += taps[i] - taps[i - 1];
    }
    const avgMs = total / (taps.length - 1);
    const tapBpm = Math.round(60000 / avgMs);
    if (tapBpm > 0 && tapBpm < 300) {
      setBpm(tapBpm);
      setBpmInput(String(tapBpm));
    }
  }, [setBpm]);

  const currentVisMode = VIS_MODES.find(m => m.key === visMode) || VIS_MODES[0];

  return (
    <div className={styles.bar}>
      {/* Advanced Visualizer */}
      <div className={styles.visRow}>
        <button
          className={styles.visModeBtn}
          onClick={cycleVisMode}
          title={`Mode: ${currentVisMode.label}`}
          aria-label={`Visualization mode: ${currentVisMode.label}`}
        >
          {currentVisMode.icon}
        </button>
        <AdvancedVisualizer bpm={bpm} positionMs={positionMs} isPlaying={isPlaying} mode={visMode} beat={beat} />
      </div>

      {/* Progress bar + Beat visualizer */}
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
        <BeatVisualizer bpm={bpm} positionMs={positionMs} isPlaying={isPlaying} />
        <div className={styles.bpmControl}>
          <Music size={14} className={styles.bpmIcon} />
          <button
            className={styles.tapBtn}
            onClick={handleTapTempo}
            title="Tap tempo"
            aria-label="Tap tempo"
          >
            <span className={styles.tapDot} />
          </button>
          <input
            type="number"
            className={styles.bpmInput}
            value={bpmInput}
            onChange={(e) => setBpmInput(e.target.value)}
            onBlur={() => {
              const val = parseInt(bpmInput, 10);
              if (!isNaN(val) && val > 0 && val < 300) setBpm(val);
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                const val = parseInt(bpmInput, 10);
                if (!isNaN(val) && val > 0 && val < 300) setBpm(val);
              }
            }}
            placeholder="BPM"
            min="1"
            max="299"
            title="Set BPM for beat visualization"
          />
          {bpm && (
            <button
              className={styles.bpmClear}
              onClick={() => { setBpm(null); setBpmInput(''); }}
              title="Clear BPM"
              aria-label="Clear BPM"
            >
              ×
            </button>
          )}
        </div>
      </div>

      {/* Controls */}
      <div className={styles.controls}>
        <button
          className={`${styles.btn} ${shuffleOn ? styles.btnActive : ''}`}
          onClick={handleShuffle}
          title={shuffleOn ? 'Shuffle on' : 'Shuffle off'}
          aria-label={shuffleOn ? 'Shuffle on' : 'Shuffle off'}
        >
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
        <button
          className={`${styles.btn} ${loopState !== 'none' ? styles.btnActive : ''}`}
          onClick={handleLoop}
          title={`Loop: ${loopState}`}
          aria-label={`Loop: ${loopState}`}
        >
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

      {/* Settings & Help */}
      <div className={styles.utilRow}>
        <button
          className={styles.utilBtn}
          onClick={onOpenHelp}
          title="Keyboard shortcuts (?)"
          aria-label="Keyboard shortcuts"
        >
          <HelpCircle size={18} />
        </button>
        <button
          className={styles.utilBtn}
          onClick={onOpenSettings}
          title="Settings"
          aria-label="Open settings"
        >
          <Settings size={18} />
        </button>
      </div>
    </div>
  );
};

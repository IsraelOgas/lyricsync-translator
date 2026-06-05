import { useEffect } from 'react';
import type { Settings } from '../types';

interface Props {
  settingsOpen: boolean;
  setSettingsOpen: (open: boolean) => void;
  settings: Settings;
  updateSetting: <K extends keyof Settings>(key: K, value: Settings[K]) => void;
  positionMs: number;
  handleTogglePlayPause: () => void;
  onOpenHelp: () => void;
}

/** Fire-and-forget POST helper — never throws. */
function post(url: string, body?: unknown): void {
  fetch(url, {
    method: 'POST',
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  }).catch(() => {});
}

const INPUT_TAGS = new Set(['INPUT', 'TEXTAREA', 'SELECT', 'BUTTON']);

export function useKeyboardShortcuts({
  settingsOpen,
  setSettingsOpen,
  settings,
  updateSetting,
  positionMs,
  handleTogglePlayPause,
  onOpenHelp,
}: Props) {
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      // Escape always works — closes settings panel
      if (e.key === 'Escape') {
        if (settingsOpen) {
          e.preventDefault();
          setSettingsOpen(false);
        }
        return;
      }

      // Gate: don't intercept when settings panel is open or user is in a form element
      if (settingsOpen) return;
      const tag = (e.target as HTMLElement)?.tagName;
      if (tag && INPUT_TAGS.has(tag)) return;

      switch (e.key) {
        case ' ':
          e.preventDefault();
          handleTogglePlayPause();
          break;

        case 'ArrowLeft':
          e.preventDefault();
          post('/api/player/seek', { position_ms: Math.max(0, positionMs - 5000) });
          break;

        case 'ArrowRight':
          e.preventDefault();
          post('/api/player/seek', { position_ms: positionMs + 5000 });
          break;

        case 'n':
        case 'N':
          e.preventDefault();
          post('/api/player/next');
          break;

        case 'p':
        case 'P':
          e.preventDefault();
          post('/api/player/previous');
          break;

        case 'ArrowUp':
          e.preventDefault();
          post('/api/player/volume', { delta: 0.05 });
          break;

        case 'ArrowDown':
          e.preventDefault();
          post('/api/player/volume', { delta: -0.05 });
          break;

        case 'm':
        case 'M':
          e.preventDefault();
          // Toggle mute: read current volume, set to 0 or restore to 0.5
          fetch('/api/player/volume')
            .then(r => r.json())
            .then(data => {
              const vol = data.volume ?? 0.5;
              post('/api/player/volume', { absolute: vol > 0 ? 0 : 0.5 });
            })
            .catch(() => {});
          break;

        case 's':
        case 'S':
          e.preventDefault();
          post('/api/player/shuffle');
          break;

        case 'l':
        case 'L':
          e.preventDefault();
          post('/api/player/loop');
          break;

        case 'r':
        case 'R':
          e.preventDefault();
          updateSetting('showRomanization', !settings.showRomanization);
          break;

        case 'c':
        case 'C':
          e.preventDefault();
          updateSetting('cinemaMode', !settings.cinemaMode);
          break;

        case '?':
          e.preventDefault();
          onOpenHelp();
          break;
      }
    }

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [settingsOpen, setSettingsOpen, settings, updateSetting, positionMs, handleTogglePlayPause, onOpenHelp]);
}
